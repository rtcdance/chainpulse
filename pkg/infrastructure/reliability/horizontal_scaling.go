package reliability

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HorizontalScaler manages horizontal scaling of services
type HorizontalScaler struct {
	mu               sync.RWMutex
	wg               sync.WaitGroup
	done             chan struct{}
	closeOnce        sync.Once
	id               string
	minInstances     int
	maxInstances     int
	currentInstances int
	targetInstances  int
	instances        map[string]*ServiceInstance
	scalingPolicy    *ScalingPolicy
	metrics          *ScalingMetrics
	lastScalingTime  time.Time
	scalingCooldown  time.Duration
}

// ServiceInstance represents a service instance
type ServiceInstance struct {
	ID              string
	Status          string // "running", "starting", "stopping", "stopped"
	CreatedAt       time.Time
	StartedAt       time.Time
	StoppedAt       time.Time
	CPUUsage        float64
	MemoryUsage     float64
	RequestCount    int64
	ErrorCount      int64
	LastHealthCheck time.Time
}

// ScalingPolicy defines scaling behavior
type ScalingPolicy struct {
	CPUThresholdUp      float64       // Scale up if CPU > this
	CPUThresholdDown    float64       // Scale down if CPU < this
	MemoryThresholdUp   float64       // Scale up if memory > this
	MemoryThresholdDown float64       // Scale down if memory < this
	RequestThreshold    int64         // Scale up if requests > this
	ScaleUpCount        int           // Number of instances to add
	ScaleDownCount      int           // Number of instances to remove
	EvaluationPeriod    time.Duration // How often to evaluate
}

// ScalingMetrics tracks scaling metrics
type ScalingMetrics struct {
	mu                  sync.RWMutex
	ScaleUpEvents       int64
	ScaleDownEvents     int64
	InstancesCreated    int64
	InstancesTerminated int64
	AverageScalingTime  time.Duration
	TotalScalingTime    time.Duration
	LastScalingTime     time.Time
}

// NewHorizontalScaler creates a new horizontal scaler
func NewHorizontalScaler(id string, minInstances, maxInstances int) *HorizontalScaler {
	return &HorizontalScaler{
		id:               id,
		minInstances:     minInstances,
		maxInstances:     maxInstances,
		currentInstances: minInstances,
		targetInstances:  minInstances,
		instances:        make(map[string]*ServiceInstance),
		scalingPolicy: &ScalingPolicy{
			CPUThresholdUp:      80.0,
			CPUThresholdDown:    20.0,
			MemoryThresholdUp:   80.0,
			MemoryThresholdDown: 20.0,
			RequestThreshold:    1000,
			ScaleUpCount:        1,
			ScaleDownCount:      1,
			EvaluationPeriod:    1 * time.Minute,
		},
		metrics:         &ScalingMetrics{},
		lastScalingTime: time.Now(),
		scalingCooldown: 5 * time.Minute,
		done:            make(chan struct{}),
	}
}

// EvaluateScaling evaluates if scaling is needed
func (hs *HorizontalScaler) EvaluateScaling(ctx context.Context) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	// Check if cooldown period has passed
	if time.Since(hs.lastScalingTime) < hs.scalingCooldown {
		return nil
	}

	// Calculate average metrics
	avgCPU, avgMemory, totalRequests := hs.calculateMetrics()

	// Determine if scaling is needed
	shouldScaleUp := false
	shouldScaleDown := false

	if avgCPU > hs.scalingPolicy.CPUThresholdUp ||
		avgMemory > hs.scalingPolicy.MemoryThresholdUp ||
		totalRequests > hs.scalingPolicy.RequestThreshold {
		shouldScaleUp = true
	}

	if avgCPU < hs.scalingPolicy.CPUThresholdDown &&
		avgMemory < hs.scalingPolicy.MemoryThresholdDown &&
		totalRequests < hs.scalingPolicy.RequestThreshold/2 {
		shouldScaleDown = true
	}

	// Perform scaling
	if shouldScaleUp {
		return hs.scaleUp(ctx)
	} else if shouldScaleDown {
		return hs.scaleDown(ctx)
	}

	return nil
}

// calculateMetrics calculates average metrics across instances
func (hs *HorizontalScaler) calculateMetrics() (float64, float64, int64) {
	if len(hs.instances) == 0 {
		return 0, 0, 0
	}

	var totalCPU, totalMemory float64
	var totalRequests int64

	for _, instance := range hs.instances {
		if instance.Status == "running" {
			totalCPU += instance.CPUUsage
			totalMemory += instance.MemoryUsage
			totalRequests += instance.RequestCount
		}
	}

	runningCount := 0
	for _, instance := range hs.instances {
		if instance.Status == "running" {
			runningCount++
		}
	}

	if runningCount == 0 {
		return 0, 0, 0
	}

	return totalCPU / float64(runningCount), totalMemory / float64(runningCount), totalRequests
}

// scaleUp adds instances
func (hs *HorizontalScaler) scaleUp(ctx context.Context) error {
	start := time.Now()
	defer func() {
		hs.recordScalingTime(time.Since(start))
	}()

	targetCount := hs.currentInstances + hs.scalingPolicy.ScaleUpCount
	if targetCount > hs.maxInstances {
		targetCount = hs.maxInstances
	}

	created := targetCount - hs.currentInstances

	for i := hs.currentInstances; i < targetCount; i++ {
		instanceID := fmt.Sprintf("%s-instance-%d", hs.id, i)
		instance := &ServiceInstance{
			ID:        instanceID,
			Status:    "starting",
			CreatedAt: time.Now(),
		}

		hs.instances[instanceID] = instance

		// Simulate instance startup
		hs.wg.Add(1)
		go func(inst *ServiceInstance) {
			defer hs.wg.Done()
			select {
			case <-hs.done:
				return // shutting down
			case <-time.After(1 * time.Second):
			}
			hs.mu.Lock()
			inst.Status = "running"
			inst.StartedAt = time.Now()
			hs.mu.Unlock()
		}(instance)
	}

	hs.currentInstances = targetCount
	hs.lastScalingTime = time.Now()

	hs.metrics.mu.Lock()
	hs.metrics.ScaleUpEvents++
	hs.metrics.InstancesCreated += int64(created)
	hs.metrics.mu.Unlock()

	return nil
}

// scaleDown removes instances
func (hs *HorizontalScaler) scaleDown(ctx context.Context) error {
	start := time.Now()
	defer func() {
		hs.recordScalingTime(time.Since(start))
	}()

	targetCount := hs.currentInstances - hs.scalingPolicy.ScaleDownCount
	if targetCount < hs.minInstances {
		targetCount = hs.minInstances
	}

	// Find instances to remove
	count := 0
	for instanceID, instance := range hs.instances {
		if count >= hs.currentInstances-targetCount {
			break
		}

		if instance.Status == "running" {
			instance.Status = "stopping"

			// Simulate graceful shutdown
			hs.wg.Add(1)
			go func(id string) {
				defer hs.wg.Done()
				select {
				case <-hs.done:
					return // shutting down
				case <-time.After(2 * time.Second):
				}
				hs.mu.Lock()
				if inst, exists := hs.instances[id]; exists {
					inst.Status = "stopped"
					inst.StoppedAt = time.Now()
					delete(hs.instances, id)
				}
				hs.mu.Unlock()
			}(instanceID)

			count++
		}
	}

	hs.currentInstances = targetCount
	hs.lastScalingTime = time.Now()

	hs.metrics.mu.Lock()
	hs.metrics.ScaleDownEvents++
	hs.metrics.InstancesTerminated += int64(count)
	hs.metrics.mu.Unlock()

	return nil
}

// recordScalingTime records scaling operation time
func (hs *HorizontalScaler) recordScalingTime(duration time.Duration) {
	hs.metrics.mu.Lock()
	defer hs.metrics.mu.Unlock()

	hs.metrics.TotalScalingTime += duration
	scalingCount := hs.metrics.ScaleUpEvents + hs.metrics.ScaleDownEvents
	if scalingCount > 0 {
		hs.metrics.AverageScalingTime = hs.metrics.TotalScalingTime / time.Duration(scalingCount)
	}
	hs.metrics.LastScalingTime = time.Now()
}

// UpdateInstanceMetrics updates metrics for an instance
func (hs *HorizontalScaler) UpdateInstanceMetrics(instanceID string, cpuUsage, memoryUsage float64, requestCount int64) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	instance, exists := hs.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found")
	}

	instance.CPUUsage = cpuUsage
	instance.MemoryUsage = memoryUsage
	instance.RequestCount = requestCount
	instance.LastHealthCheck = time.Now()

	return nil
}

// GetInstances returns all instances
func (hs *HorizontalScaler) GetInstances() []*ServiceInstance {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	instances := make([]*ServiceInstance, 0, len(hs.instances))
	for _, instance := range hs.instances {
		instances = append(instances, instance)
	}

	return instances
}

// GetMetrics returns scaling metrics
func (hs *HorizontalScaler) GetMetrics() map[string]any {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	hs.metrics.mu.RLock()
	defer hs.metrics.mu.RUnlock()

	return map[string]any{
		"current_instances":    hs.currentInstances,
		"target_instances":     hs.targetInstances,
		"min_instances":        hs.minInstances,
		"max_instances":        hs.maxInstances,
		"scale_up_events":      hs.metrics.ScaleUpEvents,
		"scale_down_events":    hs.metrics.ScaleDownEvents,
		"instances_created":    hs.metrics.InstancesCreated,
		"instances_terminated": hs.metrics.InstancesTerminated,
		"average_scaling_time": hs.metrics.AverageScalingTime.String(),
		"total_scaling_time":   hs.metrics.TotalScalingTime.String(),
		"last_scaling_time":    hs.metrics.LastScalingTime,
	}
}

// GracefulShutdown performs graceful shutdown of an instance
func (hs *HorizontalScaler) GracefulShutdown(ctx context.Context, instanceID string) error {
	hs.mu.Lock()

	instance, exists := hs.instances[instanceID]
	if !exists {
		hs.mu.Unlock()
		return fmt.Errorf("instance not found")
	}

	instance.Status = "stopping"
	hs.mu.Unlock()

	// Wait for existing connections to drain
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		// Timeout, force shutdown
	}

	hs.mu.Lock()
	instance.Status = "stopped"
	instance.StoppedAt = time.Now()
	delete(hs.instances, instanceID)
	hs.mu.Unlock()

	// Wait for in-flight goroutines to complete
	hs.wg.Wait()

	return nil
}

// GetCurrentInstanceCount returns the current number of instances
func (hs *HorizontalScaler) GetCurrentInstanceCount() int {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.currentInstances
}

// Stop signals all background goroutines to exit and waits for them to finish.
func (hs *HorizontalScaler) Stop() {
	hs.closeOnce.Do(func() {
		close(hs.done)
	})
	hs.wg.Wait()
}

// SetScalingPolicy sets the scaling policy
func (hs *HorizontalScaler) SetScalingPolicy(policy *ScalingPolicy) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.scalingPolicy = policy
}
