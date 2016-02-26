package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	tcqueue "github.com/taskcluster/taskcluster-client-go/queue"
	"github.com/taskcluster/taskcluster-client-go/tcclient"
	"github.com/taskcluster/taskcluster-worker/config"
	"github.com/taskcluster/taskcluster-worker/engines"
	"github.com/taskcluster/taskcluster-worker/plugins"
	"github.com/taskcluster/taskcluster-worker/plugins/extpoints"
	"github.com/taskcluster/taskcluster-worker/runtime"
)

// Manager is resonsible for managing the entire task lifecyle from claiming the
// task, creating a sandbox environment, and reporting the results fo the execution.
// The manager will also be responsible for ensuring tasks do not run past their max run
// time and are aborted if a cancellation message is received.
type Manager struct {
	done          chan struct{}
	interval      int
	maxCapacity   int
	engine        engines.Engine
	environment   *runtime.Environment
	pluginManager plugins.Plugin
	pluginOptions *extpoints.PluginOptions
	log           *logrus.Entry
	queue         QueueService
	provisionerId string
	workerGroup   string
	workerId      string
	sync.Mutex
	tasks map[string]*TaskRun
}

// Create a new instance of the task manager that will be responsible for claiming,
// executing, and resolving units of work (tasks).
func newTaskManager(config *config.Config, engine engines.Engine, environment *runtime.Environment, log *logrus.Entry) (*Manager, error) {
	queue := tcqueue.New(
		&tcclient.Credentials{
			ClientId:    config.Credentials.ClientId,
			AccessToken: config.Credentials.AccessToken,
			Certificate: config.Credentials.Certificate,
		},
	)
	service := &queueService{
		client:           queue,
		ProvisionerId:    config.ProvisionerId,
		WorkerGroup:      config.WorkerGroup,
		Log:              log.WithField("component", "Queue Service"),
		ExpirationOffset: config.QueueService.ExpirationOffset,
	}

	m := &Manager{
		tasks:         make(map[string]*TaskRun),
		engine:        engine,
		environment:   environment,
		interval:      config.PollingInterval,
		log:           log,
		maxCapacity:   config.Capacity,
		queue:         service,
		provisionerId: config.ProvisionerId,
		workerGroup:   config.WorkerGroup,
		workerId:      config.WorkerId,
	}

	m.pluginOptions = &extpoints.PluginOptions{
		Environment: environment,
		Engine:      &engine,
		Log:         log.WithField("component", "Plugin Manager"),
	}

	pm, err := extpoints.NewPluginManager([]string{"success"}, *m.pluginOptions)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Error creating task manager. Could not create plugin manager")
		return nil, err
	}

	m.pluginManager = pm
	return m, nil
}

// Start will initiliaze a polling cycle for tasks and spawn goroutines to
// execute units of work that has been claimed.
func (m *Manager) Start(stop <-chan struct{}, done chan struct{}) {
	m.log.Info("Polling for tasks every %d seconds\n", m.interval)
	m.done = make(chan struct{})
	doWork := time.NewTicker(time.Duration(m.interval) * time.Second)
	for {
		select {
		case <-stop:
			doWork.Stop()
			go m.Stop()
		case <-doWork.C:
			n := math.Max(float64(m.maxCapacity-len(m.tasks)), 0)
			m.claimWork(int(n))
		case <-m.done:
			close(done)
			return
		}
	}
}

func (m *Manager) Stop() {
	defer close(m.done)
	// Do interesting things
	return
}

func (m *Manager) claimWork(ntasks int) {
	if ntasks == 0 {
		return
	}

	claims := m.queue.ClaimWork(ntasks)
	for _, c := range claims {
		go m.runTask(c)
	}
}

func (m *Manager) runTask(task *TaskRun) error {
	log := m.log.WithFields(logrus.Fields{
		"taskId": task.TaskId,
		"runId":  task.RunId,
	})
	log.Info("Running Task")

	err := m.registerTask(task)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not register task")
		return err
	}

	tp := m.environment.TemporaryStorage.NewFilePath()
	task.context, task.controller, err = runtime.NewTaskContext(tp)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not create task context")
		return err
	}

	defer func() {
		err = task.controller.CloseLog()
		if err != nil {
			log.WithField("error", err.Error()).Warn("Could not properly close task log")
		}
		err = task.controller.Dispose()
		if err != nil {
			log.WithField("error", err.Error()).Warn("Could not dispose of task context")
		}
		m.deregisterTask(task)
	}()

	jsonPayload := map[string]json.RawMessage{}
	if err := json.Unmarshal(task.Definition.Payload, &jsonPayload); err != nil {
		log.WithField("error", err.Error()).Warn("Could not parse task payload")
		return err
	}

	p, err := m.engine.PayloadSchema().Parse(jsonPayload)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Payload validation failed: %s", task.Definition.Payload)
		return err
	}

	ps, err := m.pluginManager.PayloadSchema()
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not retrieve plugin payload schemas")
		return err
	}

	pluginPayload, err := ps.Parse(jsonPayload)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Plugin payload validation failed: %s", task.Definition.Payload)
		return err
	}

	popts := plugins.TaskPluginOptions{TaskInfo: &runtime.TaskInfo{}, Payload: pluginPayload}
	taskPlugins, err := m.pluginManager.NewTaskPlugin(popts)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not create task plugins")
		return err
	}

	err = taskPlugins.Prepare(task.context)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not prepare task plugins")
		return err
	}

	sandboxBuilder, err := m.engine.NewSandboxBuilder(engines.SandboxOptions{
		TaskContext: task.context,
		Payload:     p,
	})
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not create sandbox builder")
		return err
	}

	err = taskPlugins.BuildSandbox(sandboxBuilder)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not build build sandbox")
		return err
	}

	sandbox, err := sandboxBuilder.StartSandbox()
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not start sandbox")
		return err
	}

	err = taskPlugins.Started(sandbox)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not properly start sandbox")
		return err
	}

	result, err := sandbox.WaitForResult()
	if err != nil {
		log.WithField("error", err.Error()).Warn("Error when waiting for result set from sandbox")
		return err
	}

	success, err := taskPlugins.Stopped(result)
	if !success || err != nil {
		log.WithField("error", err.Error()).Warn("Could not properly stop sandbox")
		return err
	}

	err = taskPlugins.Finished(success)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not finish plugin cleanup")
		return err
	}

	err = taskPlugins.Dispose()
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not dispose plugins")
		return err
	}

	return nil
}

func (m *Manager) registerTask(task *TaskRun) error {
	name := fmt.Sprintf("%s/%d", task.TaskId, task.RunId)
	m.log.Debugf("Registered task: %s", name)

	m.Lock()
	defer m.Unlock()

	_, exists := m.tasks[name]
	if exists {
		return errors.New(fmt.Sprintf("Cannot register task %s. Task already exists.", name))
	}

	m.tasks[name] = task
	return nil
}

func (m *Manager) deregisterTask(task *TaskRun) error {
	name := fmt.Sprintf("%s/%d", task.TaskId, task.RunId)
	m.log.Debugf("Deregistered task: %s", name)

	m.Lock()
	defer m.Unlock()

	_, exists := m.tasks[name]
	if !exists {
		return errors.New(fmt.Sprintf("Cannot deregister task %s. Task does not exist", name))
	}

	delete(m.tasks, name)
	return nil
}