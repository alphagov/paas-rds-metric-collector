package scheduler

import (
	"context"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"

	"os"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/collector"
	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"github.com/alphagov/paas-rds-metric-collector/pkg/emitter"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
	"github.com/alphagov/paas-rds-metric-collector/pkg/utils"
)

const defaultRetryInterval = 1000
const defaultMaxRetries = 3

// Scheduler ...
type Scheduler struct {
	brokerinfo     brokerinfo.BrokerInfo
	metricsEmitter emitter.MetricsEmitter

	instanceRefreshInterval int

	logger lager.Logger

	metricsCollectorDrivers map[string]collector.MetricsCollectorDriver

	workers        map[workerID]*collectorWorker
	workersRunning sync.WaitGroup
	stoppedWorker  chan workerID
	cancel         context.CancelFunc
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

		metricsCollectorDrivers: map[string]collector.MetricsCollectorDriver{},
		workers:                 map[workerID]*collectorWorker{},
		stoppedWorker:           make(chan workerID, 1),

		logger: logger,
	}
}

// WithDriver ...
func (s *Scheduler) WithDriver(drivers ...collector.MetricsCollectorDriver) *Scheduler {
	for _, driver := range drivers {
		s.metricsCollectorDrivers[driver.GetName()] = driver
		s.logger.Debug("registered_driver", lager.Data{"name": driver.GetName()})
	}

	return s
}

func (s *Scheduler) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	var ctx context.Context
	ctx, s.cancel = context.WithCancel(context.Background())
	close(ready)
	s.mainLoop(ctx, signals)

	return nil
}

func (s *Scheduler) mainLoop(ctx context.Context, signals <-chan os.Signal) {
	s.logger.Info("scheduler-started")
	defer s.logger.Info("scheduler-stopped")

	timer := time.NewTimer(0)

	defer func() {
		workersStoptimeout := time.NewTimer(30 * time.Second)
		for {
			if len(s.workers) == 0 {
				return
			}
			select {
			case id := <-s.stoppedWorker:
				s.deleteWorker(id)
			case <-workersStoptimeout.C:
				s.logger.Info("timeout_waiting_for_workers", lager.Data{"instances": s.ListIntanceGUIDs()})
			}
		}
	}()

	for {
		select {
		case <-timer.C:
			timer.Reset(time.Duration(s.instanceRefreshInterval) * time.Second)

			instanceInfos, err := s.brokerinfo.ListInstances()
			if err != nil {
				s.logger.Error("unable to retreive instance guids", err)
				continue
			}

			s.logger.Debug("refresh_instances", lager.Data{"instances": instanceInfos})

			desiredWorkerIDs := map[workerID]brokerinfo.InstanceInfo{}
			for _, instanceInfo := range instanceInfos {
				for driverName, driver := range s.metricsCollectorDrivers {
					if utils.SliceContainsString(driver.SupportedTypes(), instanceInfo.Type) {
						id := workerID{Driver: driverName, InstanceGUID: instanceInfo.GUID}
						desiredWorkerIDs[id] = instanceInfo
					}
				}
			}

			for id, instanceInfo := range desiredWorkerIDs {
				if _, ok := s.workers[id]; !ok {
					s.startWorker(ctx, id, instanceInfo)
				}
			}

			for id, worker := range s.workers {
				if _, ok := desiredWorkerIDs[id]; !ok {
					worker.cancel()
				}
			}
		case id := <-s.stoppedWorker:
			s.deleteWorker(id)

		case sig := <-signals:
			s.logger.Debug("received-signal", lager.Data{"signal": sig})
			s.cancel()
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) startWorker(ctx context.Context, id workerID, instanceInfo brokerinfo.InstanceInfo) {
	workerContext, workerCancel := context.WithCancel(ctx)
	worker := &collectorWorker{
		id:             id,
		instanceInfo:   instanceInfo,
		driver:         s.metricsCollectorDrivers[id.Driver],
		metricsEmitter: s.metricsEmitter,
		retryInterval:  s.collectorRetryInterval,
		maxRetries:     s.collectorMaxRetries,
		cancel:         workerCancel,
		logger:         s.logger,
	}
	s.workers[id] = worker
	s.workersRunning.Add(1)
	go worker.run(workerContext, s.stoppedWorker)
}

func (s *Scheduler) deleteWorker(id workerID) {
	s.workersRunning.Done()
	delete(s.workers, id)
}

// Stop...
func (w *Scheduler) Stop() {
	w.cancel()
	w.workersRunning.Wait()
}

type workerID struct {
	Driver       string
	InstanceGUID string
}

type collectorWorker struct {
	id             workerID
	instanceInfo   brokerinfo.InstanceInfo
	driver         collector.MetricsCollectorDriver
	metricsEmitter emitter.MetricsEmitter
	retryInterval  int
	maxRetries     int
	cancel         context.CancelFunc
	logger         lager.Logger
	collector      collector.MetricsCollector
}

func (w *collectorWorker) run(ctx context.Context, stopped chan<- workerID) {
	defer func() { stopped <- w.id }()

	w.logger.Info("start_worker", lager.Data{
		"driver":       w.id.Driver,
		"instanceGUID": w.id.InstanceGUID,
	})

	collector, err := w.driver.NewCollector(w.instanceInfo)
	if err != nil {
		w.logger.Error("failed_creating_collector", err, lager.Data{
			"driver":       w.id.Driver,
			"instanceGUID": w.id.InstanceGUID,
		})
		return
	}

	defer collector.Close()

	defer w.logger.Info("stop_worker", lager.Data{
		"driver":       w.id.Driver,
		"instanceGUID": w.id.InstanceGUID,
	})

	timer := time.NewTimer(0)
	for {
		select {
		case <-timer.C:
			w.logger.Debug("collecting_metrics", lager.Data{
				"driver":       w.id.Driver,
				"instanceGUID": w.id.InstanceGUID,
			})
			collectedMetrics, err := collector.Collect()
			timer.Reset(time.Duration(w.driver.GetCollectInterval()) * time.Second)
			if err != nil {
				w.logger.Error("querying metrics", err, lager.Data{
					"driver":       w.id.Driver,
					"instanceGUID": w.id.InstanceGUID,
				})
				continue
			}
			w.logger.Debug("collected metrics", lager.Data{
				"driver":       w.id.Driver,
				"instanceGUID": w.id.InstanceGUID,
				"metrics":      collectedMetrics,
			})
			for _, metric := range collectedMetrics {
				w.metricsEmitter.Emit(
					metrics.MetricEnvelope{InstanceGUID: w.id.InstanceGUID, Metric: metric},
				)
			}
		case <-ctx.Done():
			return
		}
	}
}

// ListIntanceGUIDs ...
func (w *Scheduler) ListIntanceGUIDs() []string {
	instanceGUIDMap := map[string]bool{}
	for k, _ := range w.workers {
		instanceGUIDMap[k.InstanceGUID] = true
	}
	instanceGUIDs := []string{}
	for k := range instanceGUIDMap {
		instanceGUIDs = append(instanceGUIDs, k)
	}

	return instanceGUIDs
}
