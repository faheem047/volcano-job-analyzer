package analyzer

import (
    "fmt"
    "time"
    
    volcanov1alpha1 "volcano.sh/apis/pkg/apis/batch/v1alpha1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
)

// ResourceRequirements represents comprehensive resource specifications
type ResourceRequirements struct {
    CPU              resource.Quantity            `json:"cpu"`
    Memory           resource.Quantity            `json:"memory"`
    GPU              resource.Quantity            `json:"gpu"`
    Storage          resource.Quantity            `json:"storage"`
    CustomResources  map[string]resource.Quantity `json:"customResources"`
    NodeSelector     map[string]string            `json:"nodeSelector,omitempty"`
    Affinity         *corev1.Affinity             `json:"affinity,omitempty"`
    Tolerations      []corev1.Toleration          `json:"tolerations,omitempty"`
}

// TaskAnalysis provides detailed analysis of individual tasks
type TaskAnalysis struct {
    Name                string                `json:"name"`
    TaskType           TaskType              `json:"taskType"`
    Replicas           int32                 `json:"replicas"`
    Resources          ResourceRequirements  `json:"resources"`
    Dependencies       []string              `json:"dependencies"`
    CommunicationPorts []int32               `json:"communicationPorts"`
    Splittable         bool                  `json:"splittable"`
    Priority           int32                 `json:"priority"`
    EstimatedDuration  time.Duration         `json:"estimatedDuration"`
    ResourceEfficiency float64               `json:"resourceEfficiency"`
}

// TaskType categorizes different types of tasks in ML workloads
type TaskType int

const (
    TaskTypeUnknown TaskType = iota
    TaskTypeMaster
    TaskTypeWorker
    TaskTypeParameterServer
    TaskTypeDriver
    TaskTypeExecutor
    TaskTypeScheduler
    TaskTypeCoordinator
)

func (t TaskType) String() string {
    switch t {
    case TaskTypeMaster:
        return "Master"
    case TaskTypeWorker:
        return "Worker"
    case TaskTypeParameterServer:
        return "ParameterServer"
    case TaskTypeDriver:
        return "Driver"
    case TaskTypeExecutor:
        return "Executor"
    case TaskTypeScheduler:
        return "Scheduler"
    case TaskTypeCoordinator:
        return "Coordinator"
    default:
        return "Unknown"
    }
}

// JobAnalysis represents comprehensive job analysis results
type JobAnalysis struct {
    JobName              string                 `json:"jobName"`
    Namespace            string                 `json:"namespace"`
    JobType              JobType                `json:"jobType"`
    Tasks                []TaskAnalysis         `json:"tasks"`
    TotalResources       ResourceRequirements   `json:"totalResources"`
    CriticalPath         []string               `json:"criticalPath"`
    EstimatedExecutionTime time.Duration        `json:"estimatedExecutionTime"`
    ResourceUtilization  ResourceUtilization    `json:"resourceUtilization"`
    ScalabilityMetrics   ScalabilityMetrics     `json:"scalabilityMetrics"`
    NetworkRequirements  NetworkRequirements    `json:"networkRequirements"`
    SecurityConstraints  SecurityConstraints    `json:"securityConstraints"`
}

// JobType identifies the ML framework or job pattern
type JobType int

const (
    JobTypeGeneric JobType = iota
    JobTypePyTorch
    JobTypeTensorFlow
    JobTypeSpark
    JobTypeFlink
    JobTypeRay
    JobTypeMPI
)

func (j JobType) String() string {
    switch j {
    case JobTypePyTorch:
        return "PyTorch"
    case JobTypeTensorFlow:
        return "TensorFlow"
    case JobTypeSpark:
        return "Spark"
    case JobTypeFlink:
        return "Flink"
    case JobTypeRay:
        return "Ray"
    case JobTypeMPI:
        return "MPI"
    default:
        return "Generic"
    }
}

// ResourceUtilization tracks resource efficiency metrics
type ResourceUtilization struct {
    CPUEfficiency      float64 `json:"cpuEfficiency"`
    MemoryEfficiency   float64 `json:"memoryEfficiency"`
    GPUEfficiency      float64 `json:"gpuEfficiency"`
    NetworkUtilization float64 `json:"networkUtilization"`
}

// ScalabilityMetrics analyzes job scaling characteristics
type ScalabilityMetrics struct {
    MinimumNodes   int32   `json:"minimumNodes"`
    OptimalNodes   int32   `json:"optimalNodes"`
    MaximumNodes   int32   `json:"maximumNodes"`
    ScalingFactor  float64 `json:"scalingFactor"`
    BottleneckTask string  `json:"bottleneckTask"`
}

// NetworkRequirements specifies communication patterns
type NetworkRequirements struct {
    Bandwidth            resource.Quantity `json:"bandwidth"`
    Latency              time.Duration     `json:"latency"`
    CommunicationPattern string            `json:"communicationPattern"`
    RequiredPorts        []NetworkPort     `json:"requiredPorts"`
}

// NetworkPort defines network communication requirements
type NetworkPort struct {
    Port     int32  `json:"port"`
    Protocol string `json:"protocol"`
    Usage    string `json:"usage"`
}

// SecurityConstraints defines security-related requirements
type SecurityConstraints struct {
    RequiredSecurityContext *corev1.SecurityContext `json:"requiredSecurityContext,omitempty"`
    NetworkPolicies         []string                `json:"networkPolicies"`
    SecretRequirements      []string                `json:"secretRequirements"`
    ServiceAccountName      string                  `json:"serviceAccountName"`
}

// PlacementStrategy defines cluster placement recommendations
type PlacementStrategy struct {
    Strategy             PlacementType                    `json:"strategy"`
    MinimumClusters      int32                           `json:"minimumClusters"`
    RecommendedClusters  int32                           `json:"recommendedClusters"`
    ClusterRequirements  []ClusterRequirement            `json:"clusterRequirements"`
    AntiAffinityRules    []AntiAffinityRule              `json:"antiAffinityRules"`
    NetworkTopology      NetworkTopology                 `json:"networkTopology"`
    ResourceDistribution map[string]ResourceAllocation   `json:"resourceDistribution"`
}

// PlacementType defines different placement strategies
type PlacementType int

const (
    PlacementTypeConsolidated PlacementType = iota
    PlacementTypeDistributed
    PlacementTypeHybrid
    PlacementTypeGPUOptimized
    PlacementTypeLatencyOptimized
)

func (p PlacementType) String() string {
    switch p {
    case PlacementTypeConsolidated:
        return "Consolidated"
    case PlacementTypeDistributed:
        return "Distributed"
    case PlacementTypeHybrid:
        return "Hybrid"
    case PlacementTypeGPUOptimized:
        return "GPU-Optimized"
    case PlacementTypeLatencyOptimized:
        return "Latency-Optimized"
    default:
        return "Unknown"
    }
}

// ClusterRequirement specifies requirements for target clusters
type ClusterRequirement struct {
    MinimumNodes     int32                   `json:"minimumNodes"`
    RequiredLabels   map[string]string       `json:"requiredLabels"`
    RequiredTaints   []corev1.Taint          `json:"requiredTaints"`
    MinimumResources ResourceRequirements    `json:"minimumResources"`
    PreferredZone    string                  `json:"preferredZone"`
    NetworkLatency   time.Duration           `json:"networkLatency"`
}

// AntiAffinityRule defines task separation requirements
type AntiAffinityRule struct {
    TaskA     string `json:"taskA"`
    TaskB     string `json:"taskB"`
    Reason    string `json:"reason"`
    Mandatory bool   `json:"mandatory"`
}

// NetworkTopology describes network layout requirements
type NetworkTopology struct {
    RequiredBandwidth  map[string]resource.Quantity         `json:"requiredBandwidth"`
    LatencyMatrix      map[string]map[string]time.Duration  `json:"latencyMatrix"`
    CommunicationPairs []CommunicationPair                  `json:"communicationPairs"`
}

// CommunicationPair defines inter-task communication
type CommunicationPair struct {
    Source      string            `json:"source"`
    Destination string            `json:"destination"`
    Frequency   string            `json:"frequency"`
    DataVolume  resource.Quantity `json:"dataVolume"`
}

// ResourceAllocation defines how resources should be distributed
type ResourceAllocation struct {
    ClusterName string           `json:"clusterName"`
    Tasks       []TaskAllocation `json:"tasks"`
    TotalWeight float64          `json:"totalWeight"`
}

// TaskAllocation specifies task placement in clusters
type TaskAllocation struct {
    TaskName string  `json:"taskName"`
    Replicas int32   `json:"replicas"`
    Weight   float64 `json:"weight"`
}

// ClusterProfile represents cluster characteristics
type ClusterProfile struct {
    Name                 string               `json:"name"`
    AvailableResources   ResourceRequirements `json:"availableResources"`
    NetworkBandwidth     int64                `json:"networkBandwidth"` // Mbps
    NetworkLatency       time.Duration        `json:"networkLatency"`
    CostPerHour         float64              `json:"costPerHour"`
    GPUTypes            []string             `json:"gpuTypes"`
    SpecialCapabilities []string             `json:"specialCapabilities"`
    Zone                string               `json:"zone"`
    Labels              map[string]string    `json:"labels"`
}

// OptimizationResult contains optimization recommendations
type OptimizationResult struct {
    OriginalCost      float64                     `json:"originalCost"`
    OptimizedCost     float64                     `json:"optimizedCost"`
    CostSavings       float64                     `json:"costSavings"`
    PerformanceGain   float64                     `json:"performanceGain"`
    Recommendations   []OptimizationRecommendation `json:"recommendations"`
}

// OptimizationRecommendation provides specific optimization advice
type OptimizationRecommendation struct {
    Type        RecommendationType `json:"type"`
    Priority    Priority          `json:"priority"`
    Description string            `json:"description"`
    Impact      string            `json:"impact"`
    Effort      string            `json:"effort"`
}

// RecommendationType categorizes different types of recommendations
type RecommendationType int

const (
    RecommendationTypeResourceOptimization RecommendationType = iota
    RecommendationTypeNetworkOptimization
    RecommendationTypeSchedulingOptimization
    RecommendationTypeSecurityImprovement
    RecommendationTypeScalingImprovement
)

func (r RecommendationType) String() string {
    switch r {
    case RecommendationTypeResourceOptimization:
        return "Resource"
    case RecommendationTypeNetworkOptimization:
        return "Network"
    case RecommendationTypeSchedulingOptimization:
        return "Scheduling"
    case RecommendationTypeSecurityImprovement:
        return "Security"
    case RecommendationTypeScalingImprovement:
        return "Scaling"
    default:
        return "Unknown"
    }
}

// Priority defines recommendation priority levels
type Priority int

const (
    PriorityLow Priority = iota
    PriorityMedium
    PriorityHigh
    PriorityCritical
)

func (p Priority) String() string {
    switch p {
    case PriorityLow:
        return "Low"
    case PriorityMedium:
        return "Medium"
    case PriorityHigh:
        return "High"
    case PriorityCritical:
        return "Critical"
    default:
        return "Unknown"
    }
}