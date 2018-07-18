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

	workers     map[workerID]*collectorWorker
	workersLock sync.Mutex

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

func (w *Scheduler) addWorker(id workerID, worker *collectorWorker) {
	w.workersLock.Lock()
	defer w.workersLock.Unlock()
	w.workers[id] = worker
}

func (w *Scheduler) deleteWorker(id workerID) {
	w.workersLock.Lock()
	defer w.workersLock.Unlock()
	delete(w.workers, id)
}

// Start ...
func (w *Scheduler) Start() error {
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
					if !w.workerExists(id) {
						w.startWorker(id, instanceInfo)
					}
				}
			}
		}

		// Stop any instance not returned by the brokerInfo which has a worker
		for workerID, _ := range w.workers {
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
	return err
}

// Stop ...
func (w *Scheduler) Stop() {
	w.job.Quit <- true
	w.workersStarting.Wait()
	for id := range w.workers {
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
		w.addWorker(id, &collectorWorker{
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

		if w.workerExists(id) {
			err := w.workers[id].collector.Close()
			if err != nil {
				w.logger.Error("close_collector", err, lager.Data{
					"driver":       id.Driver,
					"instanceGUID": id.InstanceGUID,
				})
			}
			if w.workers[id].job != nil {
				w.workers[id].job.Quit <- true
				for {
					if !w.workers[id].job.IsRunning() {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}
		w.deleteWorker(id)
	}()
}

func (w *Scheduler) workerExists(id workerID) bool {
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
