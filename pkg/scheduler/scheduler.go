package scheduler

import (
	"sync"

	"code.cloudfoundry.org/lager"
	"github.com/carlescere/scheduler"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/collector"
	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"github.com/alphagov/paas-rds-metric-collector/pkg/emitter"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
	"github.com/alphagov/paas-rds-metric-collector/pkg/utils"
)

type workerID struct {
	Driver       string
	InstanceGUID string
}

type collectorWorker struct {
	collector collector.MetricsCollector
	job       *scheduler.Job
}

// Scheduler ...
type Scheduler struct {
	brokerinfo     brokerinfo.BrokerInfo
	metricsEmitter emitter.MetricsEmitter

	instanceRefreshInterval int
	metricCollectorInterval int

	logger lager.Logger

	metricsCollectorDrivers map[string]collector.MetricsCollectorDriver
	workers                 map[workerID]*collectorWorker
	job                     *scheduler.Job
	mux                     sync.Mutex
}

// NewScheduler ...
func NewScheduler(
	schedulerConfig config.SchedulerConfig,
	brokerInfo brokerinfo.BrokerInfo,
	metricsEmitter emitter.MetricsEmitter,
	logger lager.Logger,
) *Scheduler {

	return &Scheduler{
		brokerinfo:     brokerInfo,
		metricsEmitter: metricsEmitter,

		instanceRefreshInterval: schedulerConfig.InstanceRefreshInterval,
		metricCollectorInterval: schedulerConfig.MetricCollectorInterval,

		metricsCollectorDrivers: map[string]collector.MetricsCollectorDriver{},
		workers:                 map[workerID]*collectorWorker{},

		logger: logger,
	}
}

// WithDriver ...
func (w *Scheduler) WithDriver(drivers ...collector.MetricsCollectorDriver) *Scheduler {
	for _, driver := range drivers {
		w.metricsCollectorDrivers[driver.GetName()] = driver
	}

	return w
}

// Start ...
func (w *Scheduler) Start() error {
	var err error
	w.job, err = scheduler.Every(w.instanceRefreshInterval).Seconds().Run(func() {
		w.mux.Lock()
		defer w.mux.Unlock()

		serviceInstances, err := w.brokerinfo.ListInstanceGUIDs()
		if err != nil {
			w.logger.Error("unable to retreive instance guids", err)
			return
		}

		for _, instanceGUID := range serviceInstances {
			for driverName := range w.metricsCollectorDrivers {
				id := workerID{Driver: driverName, InstanceGUID: instanceGUID}
				if !w.WorkerExists(id) {
					w.StartWorker(id)
				}
			}
		}

		for _, instanceGUID := range w.ListIntanceGUIDs() {
			if !utils.SliceContainsString(serviceInstances, instanceGUID) {
				for driverName := range w.metricsCollectorDrivers {
					id := workerID{Driver: driverName, InstanceGUID: instanceGUID}
					w.StopWorker(id)
				}
			}
		}
	})
	return err
}

// Stop ...
func (w *Scheduler) Stop() {
	w.mux.Lock()
	defer w.mux.Unlock()
	w.job.Quit <- true
	for id := range w.workers {
		w.StopWorker(id)
	}
}

// StartWorker ...
func (w *Scheduler) StartWorker(id workerID) {
	w.logger.Info("start_worker", lager.Data{
		"driver":       id.Driver,
		"instanceGUID": id.InstanceGUID,
	})

	collector, err := w.metricsCollectorDrivers[id.Driver].NewCollector(id.InstanceGUID)
	if err != nil {
		w.logger.Error("starting worker collector", err, lager.Data{
			"driver":       id.Driver,
			"instanceGUID": id.InstanceGUID,
		})
		return
	}

	newJob, err := scheduler.Every(w.metricCollectorInterval).Seconds().Run(func() {
		w.logger.Debug("collecting metrics", lager.Data{
			"driver":       id.Driver,
			"instanceGUID": id.InstanceGUID,
		})
		collectedMetrics, err := collector.Collect()
		if err != nil {
			w.logger.Error("querying metrics", err, lager.Data{
				"driver":       id.Driver,
				"instanceGUID": id.InstanceGUID,
			})
			return
		}
		w.logger.Debug("collected metrics", lager.Data{
			"driver":       id.Driver,
			"instanceGUID": id.InstanceGUID,
			"metrics":      collectedMetrics,
		})
		for _, metric := range collectedMetrics {
			w.metricsEmitter.Emit(
				metrics.MetricEnvelope{InstanceGUID: id.InstanceGUID, Metric: metric},
			)
		}
	})
	if err != nil {
		w.logger.Error("cannot schedule the worker", err, lager.Data{
			"driver":       id.Driver,
			"instanceGUID": id.InstanceGUID,
		})
		return
	}
	w.workers[id] = &collectorWorker{
		collector: collector,
		job:       newJob,
	}
}

// StopWorker ...
func (w *Scheduler) StopWorker(id workerID) {
	w.logger.Info("stop_worker", lager.Data{
		"driver":       id.Driver,
		"instanceGUID": id.InstanceGUID,
	})

	if w.WorkerExists(id) {
		err := w.workers[id].collector.Close()
		if err != nil {
			w.logger.Error("close_collector", err, lager.Data{
				"driver":       id.Driver,
				"instanceGUID": id.InstanceGUID,
			})
		}
		w.workers[id].job.Quit <- true
	}
	delete(w.workers, id)
}

// WorkerExists ...
func (w *Scheduler) WorkerExists(id workerID) bool {
	_, ok := w.workers[id]
	return ok
}

// ListIntanceGUIDs ...
func (w *Scheduler) ListIntanceGUIDs() []string {
	instanceGUIDMap := map[string]bool{}
	for k := range w.workers {
		instanceGUIDMap[k.InstanceGUID] = true
	}
	instanceGUIDs := []string{}
	for k := range instanceGUIDMap {
		instanceGUIDs = append(instanceGUIDs, k)
	}

	return instanceGUIDs
}
