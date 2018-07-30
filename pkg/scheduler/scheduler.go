package scheduler

import (
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/carlescere/scheduler"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/collector"
	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"github.com/alphagov/paas-rds-metric-collector/pkg/emitter"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
	"github.com/alphagov/paas-rds-metric-collector/pkg/utils"
	"os"
)

type workerID struct {
	Driver       string
	InstanceGUID string
}

type collectorWorker struct {
	collector collector.MetricsCollector
	job       *scheduler.Job
}

type collectorWorkerMap struct {
	workers     map[workerID]*collectorWorker
	workersLock sync.Mutex
}

func (w *collectorWorkerMap) add(id workerID, worker *collectorWorker) {
	w.workersLock.Lock()
	defer w.workersLock.Unlock()
	w.workers[id] = worker
}

func (w *collectorWorkerMap) delete(id workerID) (*collectorWorker, bool) {
	w.workersLock.Lock()
	defer w.workersLock.Unlock()
	worker, ok := w.workers[id]
	delete(w.workers, id)
	return worker, ok
}

func (w *collectorWorkerMap) get(id workerID) (*collectorWorker, bool) {
	w.workersLock.Lock()
	defer w.workersLock.Unlock()
	worker, ok := w.workers[id]
	return worker, ok
}

func (w *collectorWorkerMap) keys() []workerID {
	w.workersLock.Lock()
	defer w.workersLock.Unlock()
	keys := make([]workerID, 0, len(w.workers))
	for k := range w.workers {
		keys = append(keys, k)
	}
	return keys
}

// Scheduler ...
type Scheduler struct {
	brokerinfo     brokerinfo.BrokerInfo
	metricsEmitter emitter.MetricsEmitter

	instanceRefreshInterval int
	metricCollectorInterval int

	logger lager.Logger

	metricsCollectorDrivers map[string]collector.MetricsCollectorDriver

	workers collectorWorkerMap

	job             *scheduler.Job
	workersStarting sync.WaitGroup
	workersStopping sync.WaitGroup
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
		workers:                 collectorWorkerMap{workers: map[workerID]*collectorWorker{}},

		logger: logger,
	}
}

// WithDriver ...
func (w *Scheduler) WithDriver(drivers ...collector.MetricsCollectorDriver) *Scheduler {
	for _, driver := range drivers {
		w.metricsCollectorDrivers[driver.GetName()] = driver
		w.logger.Debug("registered_driver", lager.Data{"name": driver.GetName()})
	}

	return w
}

func (w *Scheduler) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	var err error
	w.job, err = scheduler.Every(w.instanceRefreshInterval).Seconds().Run(func() {
		w.workersStarting.Add(1)
		defer w.workersStarting.Done()

		w.logger.Debug("refresh_instances")

		instanceInfos, err := w.brokerinfo.ListInstances()
		if err != nil {
			w.logger.Error("unable to retreive instance guids", err)
			return
		}

		w.logger.Debug("refresh_instances", lager.Data{"instances": instanceInfos})
		for _, instanceInfo := range instanceInfos {
			for driverName, driver := range w.metricsCollectorDrivers {
				if utils.SliceContainsString(driver.SupportedTypes(), instanceInfo.Type) {
					id := workerID{Driver: driverName, InstanceGUID: instanceInfo.GUID}
					if _, ok := w.workers.get(id); !ok {
						w.startWorker(id, instanceInfo)
					}
				}
			}
		}

		// Stop any instance not returned by the brokerInfo which has a worker
		for _, workerID := range w.workers.keys() {
			stillExists := false
			for _, instanceInfo := range instanceInfos {
				if instanceInfo.GUID == workerID.InstanceGUID {
					stillExists = true
					break
				}
			}
			if !stillExists {
				w.stopWorker(workerID)
			}
		}
	})
	if err != nil {
		w.logger.Debug("error-starting-scheduler")
		return err
	}
	close(ready)
	w.logger.Info("scheduler-started")
	sig := <- signals
	w.logger.Debug("received-signal", lager.Data{"signal": sig})
	w.Stop()
	return nil
}

// Stop ...
func (w *Scheduler) Stop() {
	w.job.Quit <- true
	w.workersStarting.Wait()
	for _, id := range w.workers.keys() {
		w.logger.Debug("stopping-worker")
		w.stopWorker(id)
	}
	w.workersStopping.Wait()
}

func (w *Scheduler) startWorker(id workerID, instanceInfo brokerinfo.InstanceInfo) {
	w.workersStarting.Add(1)
	go func() {
		defer w.workersStarting.Done()

		w.logger.Info("start_worker", lager.Data{
			"driver":       id.Driver,
			"instanceGUID": id.InstanceGUID,
		})

		collector, err := w.metricsCollectorDrivers[id.Driver].NewCollector(instanceInfo)
		if err != nil {
			w.logger.Error("starting worker collector", err, lager.Data{
				"driver":       id.Driver,
				"instanceGUID": id.InstanceGUID,
			})
			return
		}

		w.logger.Info("started_worker", lager.Data{
			"driver":       id.Driver,
			"instanceGUID": id.InstanceGUID,
		})

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
		w.workers.add(id, &collectorWorker{
			collector: collector,
			job:       newJob,
		})
	}()
}

func (w *Scheduler) stopWorker(id workerID) {
	w.workersStopping.Add(1)
	go func() {
		defer w.workersStopping.Done()

		w.logger.Info("stop_worker", lager.Data{
			"driver":       id.Driver,
			"instanceGUID": id.InstanceGUID,
		})

		if worker, ok := w.workers.delete(id); ok {
			err := worker.collector.Close()
			if err != nil {
				w.logger.Error("close_collector", err, lager.Data{
					"driver":       id.Driver,
					"instanceGUID": id.InstanceGUID,
				})
			}
			if worker.job != nil {
				worker.job.Quit <- true
				for {
					if !worker.job.IsRunning() {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}
	}()
}

// ListIntanceGUIDs ...
func (w *Scheduler) ListIntanceGUIDs() []string {
	instanceGUIDMap := map[string]bool{}
	for _, k := range w.workers.keys() {
		instanceGUIDMap[k.InstanceGUID] = true
	}
	instanceGUIDs := []string{}
	for k := range instanceGUIDMap {
		instanceGUIDs = append(instanceGUIDs, k)
	}

	return instanceGUIDs
}
