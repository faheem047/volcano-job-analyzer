package analyzer

import (
    "fmt"
    "strings"
    "time"

    volcanov1alpha1 "volcano.sh/apis/pkg/apis/batch/v1alpha1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
)

// AnalyzerConfig holds configuration for the analyzer
type AnalyzerConfig struct {
    EnableAdvancedMetrics       bool
    NetworkLatencyThreshold     time.Duration
    ResourceEfficiencyThreshold float64
    DefaultJobTimeout           time.Duration
}

// Analyzer performs comprehensive analysis of Volcano Jobs
type Analyzer struct {
    config *AnalyzerConfig
}

// NewAnalyzer creates a new analyzer instance
func NewAnalyzer(config *AnalyzerConfig) *Analyzer {
    return &Analyzer{
        config: config,
    }
}

// AnalyzeJob performs comprehensive analysis of a Volcano Job
func (a *Analyzer) AnalyzeJob(job *volcanov1alpha1.Job) (*JobAnalysis, error) {
    analysis := &JobAnalysis{
        JobName:   job.Name,
        Namespace: job.Namespace,
        JobType:   a.identifyJobType(job),
        Tasks:     make([]TaskAnalysis, 0, len(job.Spec.Tasks)),
    }

    // Analyze individual tasks
    for _, task := range job.Spec.Tasks {
        taskAnalysis, err := a.analyzeTask(task, job)
        if err != nil {
            return nil, fmt.Errorf("failed to analyze task %s: %w", task.Name, err)
        }
        analysis.Tasks = append(analysis.Tasks, *taskAnalysis)
    }

    // Calculate aggregate metrics
    a.calculateAggregateResources(analysis)
    a.analyzeCriticalPath(analysis)
    a.calculateResourceUtilization(analysis)
    a.analyzeScalabilityMetrics(analysis)
    a.analyzeNetworkRequirements(analysis)
    a.analyzeSecurityConstraints(analysis, job)
    a.estimateExecutionTime(analysis)

    return analysis, nil
}

// analyzeTask performs detailed analysis of a single task
func (a *Analyzer) analyzeTask(task volcanov1alpha1.TaskSpec, job *volcanov1alpha1.Job) (*TaskAnalysis, error) {
    if len(task.Template.Spec.Containers) == 0 {
        return nil, fmt.Errorf("task %s has no containers", task.Name)
    }

    container := task.Template.Spec.Containers[0]
    
    taskAnalysis := &TaskAnalysis{
        Name:         task.Name,
        TaskType:     a.classifyTaskType(task.Name),
        Replicas:     task.Replicas,
        Dependencies: a.extractDependencies(task, job),
        Priority:     a.calculateTaskPriority(task),
    }

    // Extract resource requirements
    taskAnalysis.Resources = a.extractResourceRequirements(container, task.Template.Spec)
    
    // Analyze task characteristics
    taskAnalysis.Splittable = a.isTaskSplittable(taskAnalysis.TaskType, task)
    taskAnalysis.EstimatedDuration = a.estimateTaskDuration(task, taskAnalysis.TaskType)
    taskAnalysis.ResourceEfficiency = a.calculateResourceEfficiency(taskAnalysis.Resources)
    taskAnalysis.CommunicationPorts = a.extractCommunicationPorts(container)

    return taskAnalysis, nil
}

// extractResourceRequirements extracts and normalizes resource requirements
func (a *Analyzer) extractResourceRequirements(container corev1.Container, podSpec corev1.PodSpec) ResourceRequirements {
    resources := ResourceRequirements{
        CustomResources: make(map[string]resource.Quantity),
    }

    if container.Resources.Requests != nil {
        if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
            resources.CPU = cpu
        }
        if memory, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
            resources.Memory = memory
        }
        if storage, ok := container.Resources.Requests[corev1.ResourceStorage]; ok {
            resources.Storage = storage
        }
        if gpu, ok := container.Resources.Requests["nvidia.com/gpu"]; ok {
            resources.GPU = gpu
        }

        // Extract custom resources
        for resourceName, quantity := range container.Resources.Requests {
            if !a.isStandardResource(resourceName) {
                resources.CustomResources[string(resourceName)] = quantity
            }
        }
    }

    // Extract node selector
    if podSpec.NodeSelector != nil {
        resources.NodeSelector = podSpec.NodeSelector
    }

    // Extract affinity
    if podSpec.Affinity != nil {
        resources.Affinity = podSpec.Affinity
    }

    // Extract tolerations
    if podSpec.Tolerations != nil {
        resources.Tolerations = podSpec.Tolerations
    }

    return resources
}

// classifyTaskType determines the type of task based on naming patterns and characteristics
func (a *Analyzer) classifyTaskType(taskName string) TaskType {
    taskLower := strings.ToLower(taskName)
    
    patterns := map[TaskType][]string{
        TaskTypeMaster:          {"master", "chief", "coordinator", "leader"},
        TaskTypeWorker:          {"worker", "slave", "node"},
        TaskTypeExecutor:        {"executor"},
        TaskTypeParameterServer: {"ps", "parameter-server", "param-server"},
        TaskTypeDriver:          {"driver", "client"},
        TaskTypeScheduler:       {"scheduler", "dispatcher"},
    }

    for taskType, keywords := range patterns {
        for _, keyword := range keywords {
            if strings.Contains(taskLower, keyword) {
                return taskType
            }
        }
    }

    return TaskTypeUnknown
}

// identifyJobType identifies the ML framework or job pattern
func (a *Analyzer) identifyJobType(job *volcanov1alpha1.Job) JobType {
    // Check annotations and labels for framework hints
    if annotations := job.GetAnnotations(); annotations != nil {
        if framework, ok := annotations["volcano.sh/framework"]; ok {
            switch strings.ToLower(framework) {
            case "pytorch":
                return JobTypePyTorch
            case "tensorflow":
                return JobTypeTensorFlow
            case "spark":
                return JobTypeSpark
            case "flink":
                return JobTypeFlink
            case "ray":
                return JobTypeRay
            case "mpi":
                return JobTypeMPI
            }
        }
    }

    // Infer from task patterns
    taskNames := make([]string, 0, len(job.Spec.Tasks))
    for _, task := range job.Spec.Tasks {
        taskNames = append(taskNames, strings.ToLower(task.Name))
    }

    // PyTorch pattern: master + worker
    if a.containsAll(taskNames, []string{"master"}) && a.containsAny(taskNames, []string{"worker"}) {
        return JobTypePyTorch
    }

    // TensorFlow pattern: chief + worker + ps
    if a.containsAny(taskNames, []string{"chief", "ps", "parameter"}) {
        return JobTypeTensorFlow
    }

    // Spark pattern: driver + executor
    if a.containsAll(taskNames, []string{"driver"}) && a.containsAny(taskNames, []string{"executor"}) {
        return JobTypeSpark
    }

    return JobTypeGeneric
}

// calculateAggregateResources calculates total resource requirements
func (a *Analyzer) calculateAggregateResources(analysis *JobAnalysis) {
    totalResources := ResourceRequirements{
        CustomResources: make(map[string]resource.Quantity),
    }

    for _, task := range analysis.Tasks {
        // Multiply task resources by replica count
        cpuTotal := resource.NewMilliQuantity(0, resource.DecimalSI)
        if !task.Resources.CPU.IsZero() {
            cpuTotal.Add(task.Resources.CPU)
        }
        if task.Replicas > 1 {
            // Quantity lacks Mul in this k8s version; add repeatedly
            for i := int32(1); i < task.Replicas; i++ {
                cpuTotal.Add(task.Resources.CPU)
            }
        }
        totalResources.CPU.Add(*cpuTotal)

        memTotal := resource.NewQuantity(0, resource.BinarySI)
        if !task.Resources.Memory.IsZero() {
            memTotal.Add(task.Resources.Memory)
        }
        if task.Replicas > 1 {
            for i := int32(1); i < task.Replicas; i++ {
                memTotal.Add(task.Resources.Memory)
            }
        }
        totalResources.Memory.Add(*memTotal)

        gpuTotal := resource.NewQuantity(0, resource.DecimalSI)
        if !task.Resources.GPU.IsZero() {
            gpuTotal.Add(task.Resources.GPU)
        }
        if task.Replicas > 1 {
            for i := int32(1); i < task.Replicas; i++ {
                gpuTotal.Add(task.Resources.GPU)
            }
        }
        totalResources.GPU.Add(*gpuTotal)

        storageTotal := resource.NewQuantity(0, resource.BinarySI)
        if !task.Resources.Storage.IsZero() {
            storageTotal.Add(task.Resources.Storage)
        }
        if task.Replicas > 1 {
            for i := int32(1); i < task.Replicas; i++ {
                storageTotal.Add(task.Resources.Storage)
            }
        }
        totalResources.Storage.Add(*storageTotal)

        // Aggregate custom resources
        for resourceName, quantity := range task.Resources.CustomResources {
            total := resource.NewQuantity(0, quantity.Format)
            total.Add(quantity)
            if task.Replicas > 1 {
                for i := int32(1); i < task.Replicas; i++ {
                    total.Add(quantity)
                }
            }
            if existing, ok := totalResources.CustomResources[resourceName]; ok {
                existing.Add(*total)
                totalResources.CustomResources[resourceName] = existing
            } else {
                totalResources.CustomResources[resourceName] = *total
            }
        }
    }

    analysis.TotalResources = totalResources
}

// analyzeCriticalPath identifies the critical execution path
func (a *Analyzer) analyzeCriticalPath(analysis *JobAnalysis) {
    criticalPath := make([]string, 0)
    
    // Simple heuristic: tasks that cannot be split and have dependencies
    for _, task := range analysis.Tasks {
        if !task.Splittable || len(task.Dependencies) > 0 {
            criticalPath = append(criticalPath, task.Name)
        }
    }

    // Add bottleneck tasks (high resource requirements)
    for _, task := range analysis.Tasks {
        if task.ResourceEfficiency < a.config.ResourceEfficiencyThreshold {
            criticalPath = append(criticalPath, task.Name)
        }
    }

    analysis.CriticalPath = criticalPath
}

// calculateResourceUtilization estimates resource utilization efficiency
func (a *Analyzer) calculateResourceUtilization(analysis *JobAnalysis) {
    // Calculate based on task types and typical utilization patterns
    cpuEfficiency := 0.8  // Default 80% efficiency
    memoryEfficiency := 0.9
    gpuEfficiency := 0.95
    networkUtilization := 0.6

    // Adjust based on job characteristics
    if analysis.JobType == JobTypeTensorFlow || analysis.JobType == JobTypePyTorch {
        gpuEfficiency = 0.9  // GPU-intensive workloads
        networkUtilization = 0.8  // High network utilization for distributed training
    }

    // Adjust based on task distribution
    workerTasks := 0
    for _, task := range analysis.Tasks {
        if task.TaskType == TaskTypeWorker {
            workerTasks++
        }
    }

    if workerTasks > 4 {
        // Large distributed jobs may have coordination overhead
        cpuEfficiency *= 0.95
        networkUtilization *= 1.1
    }

    analysis.ResourceUtilization = ResourceUtilization{
        CPUEfficiency:      cpuEfficiency,
        MemoryEfficiency:   memoryEfficiency,
        GPUEfficiency:      gpuEfficiency,
        NetworkUtilization: networkUtilization,
    }
}

// analyzeScalabilityMetrics calculates scaling characteristics
func (a *Analyzer) analyzeScalabilityMetrics(analysis *JobAnalysis) {
    minNodes := int32(1)
    optimalNodes := int32(1)
    maxNodes := int32(1)
    var bottleneckTask string

    for _, task := range analysis.Tasks {
        if !task.Splittable {
            minNodes = maxInt32(minNodes, task.Replicas)
        }
        
        optimalNodes += task.Replicas
        maxNodes += task.Replicas * 2  // Assume 2x scaling potential

        // Identify bottleneck (non-splittable tasks with high resource requirements)
        if !task.Splittable && task.ResourceEfficiency < 0.8 {
            bottleneckTask = task.Name
        }
    }

    scalingFactor := float64(maxNodes) / float64(minNodes)

    analysis.ScalabilityMetrics = ScalabilityMetrics{
        MinimumNodes:   minNodes,
        OptimalNodes:   optimalNodes,
        MaximumNodes:   maxNodes,
        ScalingFactor:  scalingFactor,
        BottleneckTask: bottleneckTask,
    }
}

// analyzeNetworkRequirements analyzes communication patterns
func (a *Analyzer) analyzeNetworkRequirements(analysis *JobAnalysis) {
    bandwidth := resource.MustParse("1Gi")  // Default 1 Gbps
    latency := 10 * time.Millisecond        // Default 10ms

    pattern := "point-to-point"
    if analysis.JobType == JobTypeTensorFlow || analysis.JobType == JobTypePyTorch {
        pattern = "all-reduce"
        bandwidth = resource.MustParse("10Gi")  // Higher bandwidth for ML training
        latency = 5 * time.Millisecond          // Lower latency requirement
    }

    ports := make([]NetworkPort, 0)
    for _, task := range analysis.Tasks {
        for _, port := range task.CommunicationPorts {
            ports = append(ports, NetworkPort{
                Port:     port,
                Protocol: "TCP",
                Usage:    fmt.Sprintf("%s communication", task.Name),
            })
        }
    }

    analysis.NetworkRequirements = NetworkRequirements{
        Bandwidth:            bandwidth,
        Latency:              latency,
        CommunicationPattern: pattern,
        RequiredPorts:        ports,
    }
}

// analyzeSecurityConstraints extracts security requirements
func (a *Analyzer) analyzeSecurityConstraints(analysis *JobAnalysis, job *volcanov1alpha1.Job) {
    constraints := SecurityConstraints{
        NetworkPolicies:    make([]string, 0),
        SecretRequirements: make([]string, 0),
    }

    // Extract from job specification
    if len(job.Spec.Tasks) > 0 {
        template := job.Spec.Tasks[0].Template
        if template.Spec.ServiceAccountName != "" {
            constraints.ServiceAccountName = template.Spec.ServiceAccountName
        }

        if template.Spec.SecurityContext != nil {
            constraints.RequiredSecurityContext = template.Spec.SecurityContext
        }

        // Check for secret volumes
        for _, volume := range template.Spec.Volumes {
            if volume.Secret != nil {
                constraints.SecretRequirements = append(constraints.SecretRequirements, volume.Secret.SecretName)
            }
        }
    }

    analysis.SecurityConstraints = constraints
}

// estimateExecutionTime calculates estimated job execution time
func (a *Analyzer) estimateExecutionTime(analysis *JobAnalysis) {
    maxDuration := time.Duration(0)
    
    for _, task := range analysis.Tasks {
        if task.EstimatedDuration > maxDuration {
            maxDuration = task.EstimatedDuration
        }
    }

    // Add overhead for coordination and setup
    overhead := maxDuration / 10  // 10% overhead
    analysis.EstimatedExecutionTime = maxDuration + overhead
}

// Helper functions

func (a *Analyzer) isStandardResource(resourceName corev1.ResourceName) bool {
    standardResources := []corev1.ResourceName{
        corev1.ResourceCPU,
        corev1.ResourceMemory,
        corev1.ResourceStorage,
        corev1.ResourceEphemeralStorage,
    }

    for _, standard := range standardResources {
        if resourceName == standard {
            return true
        }
    }

    return strings.HasPrefix(string(resourceName), "nvidia.com/") ||
           strings.HasPrefix(string(resourceName), "amd.com/")
}

func (a *Analyzer) containsAll(slice []string, items []string) bool {
    for _, item := range items {
        found := false
        for _, s := range slice {
            if strings.Contains(s, item) {
                found = true
                break
            }
        }
        if !found {
            return false
        }
    }
    return true
}

func (a *Analyzer) containsAny(slice []string, items []string) bool {
    for _, item := range items {
        for _, s := range slice {
            if strings.Contains(s, item) {
                return true
            }
        }
    }
    return false
}

func (a *Analyzer) extractDependencies(task volcanov1alpha1.TaskSpec, job *volcanov1alpha1.Job) []string {
    dependencies := make([]string, 0)
    
    // Check for init containers that might indicate dependencies
    if len(task.Template.Spec.InitContainers) > 0 {
        for _, initContainer := range task.Template.Spec.InitContainers {
            // Simple heuristic: if init container name contains another task name
            for _, otherTask := range job.Spec.Tasks {
                if strings.Contains(initContainer.Name, otherTask.Name) && otherTask.Name != task.Name {
                    dependencies = append(dependencies, otherTask.Name)
                }
            }
        }
    }

    // Check environment variables for dependencies
    for _, container := range task.Template.Spec.Containers {
        for _, env := range container.Env {
            for _, otherTask := range job.Spec.Tasks {
                if strings.Contains(env.Value, otherTask.Name) && otherTask.Name != task.Name {
                    dependencies = append(dependencies, otherTask.Name)
                }
            }
        }
    }

    return dependencies
}

func (a *Analyzer) calculateTaskPriority(task volcanov1alpha1.TaskSpec) int32 {
    priority := int32(0)
    
    if task.Template.Spec.Priority != nil {
        priority = *task.Template.Spec.Priority
    }

    return priority
}

func (a *Analyzer) isTaskSplittable(taskType TaskType, task volcanov1alpha1.TaskSpec) bool {
    // Tasks that typically cannot be split
    nonSplittable := []TaskType{
        TaskTypeMaster,
        TaskTypeDriver,
        TaskTypeCoordinator,
    }

    for _, nt := range nonSplittable {
        if taskType == nt {
            return false
        }
    }

    // Check for specific annotations that indicate splittability
    if annotations := task.Template.GetAnnotations(); annotations != nil {
        if splittable, ok := annotations["volcano.sh/splittable"]; ok {
            return splittable == "true"
        }
    }

    return true
}

func (a *Analyzer) estimateTaskDuration(task volcanov1alpha1.TaskSpec, taskType TaskType) time.Duration {
    // Default duration estimates based on task type
    defaultDurations := map[TaskType]time.Duration{
        TaskTypeMaster:          2 * time.Hour,
        TaskTypeWorker:          1 * time.Hour,
        TaskTypeParameterServer: 3 * time.Hour,
        TaskTypeDriver:          30 * time.Minute,
    }

    if duration, ok := defaultDurations[taskType]; ok {
        return duration
    }

    return a.config.DefaultJobTimeout
}

func (a *Analyzer) calculateResourceEfficiency(resources ResourceRequirements) float64 {
    efficiency := 0.8  // Default efficiency

    // Adjust based on resource ratios and patterns
    if !resources.CPU.IsZero() && !resources.Memory.IsZero() {
        // Calculate CPU to memory ratio and adjust efficiency
        cpuValue := float64(resources.CPU.MilliValue())
        memoryValue := float64(resources.Memory.Value())
        
        ratio := cpuValue / (memoryValue / (1024 * 1024 * 1024)) // CPU millicores per GB memory
        
        // Optimal ratio is around 1000-2000 millicores per GB
        if ratio >= 1000 && ratio <= 2000 {
            efficiency = 0.95
        } else if ratio < 500 || ratio > 4000 {
            efficiency = 0.6
        }
    }

    return efficiency
}

func (a *Analyzer) extractCommunicationPorts(container corev1.Container) []int32 {
    ports := make([]int32, 0)
    
    for _, port := range container.Ports {
        ports = append(ports, port.ContainerPort)
    }

    return ports
}

func maxInt32(a, b int32) int32 {
    if a > b {
        return a
    }
    return b
}