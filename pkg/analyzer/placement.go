package analyzer

import (
    "fmt"
    "math"
    "sort"
    "time"

    "k8s.io/apimachinery/pkg/api/resource"
)

// PlacementAnalyzer provides intelligent cluster placement recommendations
type PlacementAnalyzer struct {
    config *PlacementConfig
}

// PlacementConfig holds configuration for placement analysis
type PlacementConfig struct {
    MaxClusters                   int
    MinClusterUtilization        float64
    MaxClusterUtilization        float64
    NetworkLatencyWeight         float64
    ResourceAffinityWeight       float64
    CostOptimizationWeight       float64
    PerformanceOptimizationWeight float64
}

// NewPlacementAnalyzer creates a new placement analyzer
func NewPlacementAnalyzer(config *PlacementConfig) *PlacementAnalyzer {
    return &PlacementAnalyzer{config: config}
}

// GeneratePlacementStrategy creates optimal placement recommendations
func (p *PlacementAnalyzer) GeneratePlacementStrategy(analysis *JobAnalysis, clusterProfiles []ClusterProfile) (*PlacementStrategy, error) {
    if len(clusterProfiles) == 0 {
        return nil, fmt.Errorf("no cluster profiles provided")
    }

    strategy := &PlacementStrategy{
        ClusterRequirements:  make([]ClusterRequirement, 0),
        AntiAffinityRules:   make([]AntiAffinityRule, 0),
        ResourceDistribution: make(map[string]ResourceAllocation),
    }

    // Determine optimal placement strategy type
    strategy.Strategy = p.determineOptimalStrategy(analysis)
    
    // Calculate cluster requirements
    strategy.MinimumClusters = p.calculateMinimumClusters(analysis)
    strategy.RecommendedClusters = p.calculateOptimalClusters(analysis, clusterProfiles)

    // Generate cluster requirements
    strategy.ClusterRequirements = p.generateClusterRequirements(analysis)

    // Create anti-affinity rules
    strategy.AntiAffinityRules = p.generateAntiAffinityRules(analysis)

    // Generate resource distribution plan
    distribution, err := p.generateResourceDistribution(analysis, clusterProfiles, strategy)
    if err != nil {
        return nil, fmt.Errorf("failed to generate resource distribution: %w", err)
    }
    strategy.ResourceDistribution = distribution

    // Analyze network topology requirements
    strategy.NetworkTopology = p.analyzeNetworkTopology(analysis)

    return strategy, nil
}

// determineOptimalStrategy selects the best placement strategy
func (p *PlacementAnalyzer) determineOptimalStrategy(analysis *JobAnalysis) PlacementType {
    // GPU-intensive workloads benefit from GPU-optimized placement
    if !analysis.TotalResources.GPU.IsZero() {
        gpuValue, _ := analysis.TotalResources.GPU.AsInt64()
        if gpuValue > 4 {
            return PlacementTypeGPUOptimized
        }
    }

    // High communication workloads need latency optimization
    if analysis.NetworkRequirements.CommunicationPattern == "all-reduce" ||
       analysis.NetworkRequirements.Latency < 10*time.Millisecond {
        return PlacementTypeLatencyOptimized
    }

    // Large-scale workloads benefit from distribution
    totalReplicas := int32(0)
    for _, task := range analysis.Tasks {
        totalReplicas += task.Replicas
    }

    if totalReplicas > 10 {
        return PlacementTypeDistributed
    }

    // Small workloads can be consolidated
    if totalReplicas <= 5 {
        return PlacementTypeConsolidated
    }

    return PlacementTypeHybrid
}

// calculateMinimumClusters determines the minimum number of clusters needed
func (p *PlacementAnalyzer) calculateMinimumClusters(analysis *JobAnalysis) int32 {
    minClusters := int32(1)

    // Count non-splittable tasks that require separate clusters
    for _, task := range analysis.Tasks {
        if !task.Splittable && task.TaskType == TaskTypeMaster {
            minClusters = maxInt32(minClusters, 1)
        }
    }

    // Consider resource constraints
    if !analysis.TotalResources.GPU.IsZero() {
        gpuValue, _ := analysis.TotalResources.GPU.AsInt64()
        // Assume max 8 GPUs per cluster
        minClustersForGPU := int32(math.Ceil(float64(gpuValue) / 8.0))
        minClusters = maxInt32(minClusters, minClustersForGPU)
    }

    return minClusters
}

// calculateOptimalClusters determines the optimal number of clusters
func (p *PlacementAnalyzer) calculateOptimalClusters(analysis *JobAnalysis, profiles []ClusterProfile) int32 {
    minClusters := p.calculateMinimumClusters(analysis)
    
    // Performance-based calculation
    optimalClusters := minClusters

    // For distributed training, more clusters can improve performance
    if analysis.JobType == JobTypeTensorFlow || analysis.JobType == JobTypePyTorch {
        workerCount := int32(0)
        for _, task := range analysis.Tasks {
            if task.TaskType == TaskTypeWorker {
                workerCount += task.Replicas
            }
        }
        
        // Optimal: 2-4 workers per cluster for distributed training
        if workerCount > 4 {
            optimalClusters = maxInt32(optimalClusters, workerCount/3)
        }
    }

    // Limit by available clusters
    maxAvailable := int32(len(profiles))
    if optimalClusters > maxAvailable {
        optimalClusters = maxAvailable
    }

    // Limit by configuration
    if optimalClusters > int32(p.config.MaxClusters) {
        optimalClusters = int32(p.config.MaxClusters)
    }

    return optimalClusters
}

// generateClusterRequirements creates specific requirements for target clusters
func (p *PlacementAnalyzer) generateClusterRequirements(analysis *JobAnalysis) []ClusterRequirement {
    requirements := make([]ClusterRequirement, 0)

    // Base requirement for any cluster
    baseReq := ClusterRequirement{
        MinimumNodes:     1,
        RequiredLabels:   make(map[string]string),
        MinimumResources: ResourceRequirements{},
        NetworkLatency:   analysis.NetworkRequirements.Latency,
    }

    // GPU requirements
    if !analysis.TotalResources.GPU.IsZero() {
        baseReq.RequiredLabels["accelerator"] = "nvidia-tesla-v100"
        baseReq.MinimumResources.GPU = analysis.TotalResources.GPU
    }

    // High-memory requirements
    memoryValue := analysis.TotalResources.Memory.Value()
    if memoryValue > 64*1024*1024*1024 { // > 64GB
        baseReq.RequiredLabels["memory-type"] = "high-memory"
    }

    // Network requirements
    if analysis.NetworkRequirements.CommunicationPattern == "all-reduce" {
        baseReq.RequiredLabels["network"] = "high-bandwidth"
        baseReq.NetworkLatency = 5 * time.Millisecond
    }

    requirements = append(requirements, baseReq)

    return requirements
}

// generateAntiAffinityRules creates task separation rules
func (p *PlacementAnalyzer) generateAntiAffinityRules(analysis *JobAnalysis) []AntiAffinityRule {
    rules := make([]AntiAffinityRule, 0)

    // Find master/coordinator tasks
    masters := make([]string, 0)
    workers := make([]string, 0)

    for _, task := range analysis.Tasks {
        switch task.TaskType {
        case TaskTypeMaster, TaskTypeCoordinator, TaskTypeDriver:
            masters = append(masters, task.Name)
        case TaskTypeWorker, TaskTypeExecutor:
            workers = append(workers, task.Name)
        }
    }

    // Masters should be on different clusters from each other
    for i := 0; i < len(masters); i++ {
        for j := i + 1; j < len(masters); j++ {
            rules = append(rules, AntiAffinityRule{
                TaskA:     masters[i],
                TaskB:     masters[j],
                Reason:    "High availability - separate master components",
                Mandatory: true,
            })
        }
    }

    // High-resource tasks should be distributed
    for _, task := range analysis.Tasks {
        if task.ResourceEfficiency < 0.7 && task.Replicas > 1 {
            rules = append(rules, AntiAffinityRule{
                TaskA:     task.Name,
                TaskB:     task.Name,
                Reason:    "Resource optimization - distribute replicas",
                Mandatory: false,
            })
        }
    }

    return rules
}

// generateResourceDistribution creates optimal resource allocation plan
func (p *PlacementAnalyzer) generateResourceDistribution(analysis *JobAnalysis, profiles []ClusterProfile, strategy *PlacementStrategy) (map[string]ResourceAllocation, error) {

    // Sort clusters by capacity and suitability
    rankedClusters := p.rankClusters(profiles, analysis)
    
    switch strategy.Strategy {
    case PlacementTypeConsolidated:
        return p.generateConsolidatedDistribution(analysis, rankedClusters)
    case PlacementTypeDistributed:
        return p.generateDistributedDistribution(analysis, rankedClusters)
    case PlacementTypeGPUOptimized:
        return p.generateGPUOptimizedDistribution(analysis, rankedClusters)
    case PlacementTypeLatencyOptimized:
        return p.generateLatencyOptimizedDistribution(analysis, rankedClusters)
    case PlacementTypeHybrid:
        return p.generateHybridDistribution(analysis, rankedClusters)
    default:
        return p.generateDistributedDistribution(analysis, rankedClusters)
    }
}

// rankClusters sorts clusters by suitability for the workload
func (p *PlacementAnalyzer) rankClusters(profiles []ClusterProfile, analysis *JobAnalysis) []ClusterProfile {
    scored := make([]struct {
        profile ClusterProfile
        score   float64
    }, len(profiles))

    for i, profile := range profiles {
        score := p.calculateClusterScore(profile, analysis)
        scored[i] = struct {
            profile ClusterProfile
            score   float64
        }{profile, score}
    }

    // Sort by score (descending)
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].score > scored[j].score
    })

    result := make([]ClusterProfile, len(profiles))
    for i, s := range scored {
        result[i] = s.profile
    }

    return result
}

// calculateClusterScore computes suitability score for a cluster
func (p *PlacementAnalyzer) calculateClusterScore(profile ClusterProfile, analysis *JobAnalysis) float64 {
    score := 0.0

    // Resource availability score
    resourceScore := p.calculateResourceScore(profile, analysis)
    score += resourceScore * p.config.ResourceAffinityWeight

    // Network performance score
    networkScore := p.calculateNetworkScore(profile, analysis)
    score += networkScore * p.config.NetworkLatencyWeight

    // Cost efficiency score
    costScore := p.calculateCostScore(profile, analysis)
    score += costScore * p.config.CostOptimizationWeight

    // Performance optimization score
    performanceScore := p.calculatePerformanceScore(profile, analysis)
    score += performanceScore * p.config.PerformanceOptimizationWeight

    return score
}

// Implementation of specific distribution strategies
func (p *PlacementAnalyzer) generateConsolidatedDistribution(analysis *JobAnalysis, clusters []ClusterProfile) (map[string]ResourceAllocation, error) {
    distribution := make(map[string]ResourceAllocation)
    
    if len(clusters) == 0 {
        return nil, fmt.Errorf("no suitable clusters available")
    }

    // Use the best cluster for everything
    bestCluster := clusters[0]
    
    allocation := ResourceAllocation{
        ClusterName: bestCluster.Name,
        Tasks:       make([]TaskAllocation, 0),
        TotalWeight: 1.0,
    }

    for _, task := range analysis.Tasks {
        allocation.Tasks = append(allocation.Tasks, TaskAllocation{
            TaskName: task.Name,
            Replicas: task.Replicas,
            Weight:   1.0,
        })
    }

    distribution[bestCluster.Name] = allocation
    return distribution, nil
}

func (p *PlacementAnalyzer) generateDistributedDistribution(analysis *JobAnalysis, clusters []ClusterProfile) (map[string]ResourceAllocation, error) {
    distribution := make(map[string]ResourceAllocation)
    
    clusterCount := len(clusters)
    if clusterCount == 0 {
        return nil, fmt.Errorf("no suitable clusters available")
    }

    // Distribute tasks across available clusters
    for i, cluster := range clusters {
        allocation := ResourceAllocation{
            ClusterName: cluster.Name,
            Tasks:       make([]TaskAllocation, 0),
            TotalWeight: 1.0 / float64(clusterCount),
        }

        for _, task := range analysis.Tasks {
            if task.Splittable {
                // Distribute replicas across clusters
                replicasPerCluster := task.Replicas / int32(clusterCount)
                if i == 0 {
                    replicasPerCluster += task.Replicas % int32(clusterCount) // Give remainder to first cluster
                }
                
                if replicasPerCluster > 0 {
                    allocation.Tasks = append(allocation.Tasks, TaskAllocation{
                        TaskName: task.Name,
                        Replicas: replicasPerCluster,
                        Weight:   float64(replicasPerCluster) / float64(task.Replicas),
                    })
                }
            } else if i == 0 {
                // Non-splittable tasks go to the first (best) cluster
                allocation.Tasks = append(allocation.Tasks, TaskAllocation{
                    TaskName: task.Name,
                    Replicas: task.Replicas,
                    Weight:   1.0,
                })
            }
        }

        if len(allocation.Tasks) > 0 {
            distribution[cluster.Name] = allocation
        }
    }

    return distribution, nil
}

func (p *PlacementAnalyzer) generateGPUOptimizedDistribution(analysis *JobAnalysis, clusters []ClusterProfile) (map[string]ResourceAllocation, error) {
    // Filter clusters with GPU capabilities
    gpuClusters := make([]ClusterProfile, 0)
    for _, cluster := range clusters {
        if len(cluster.GPUTypes) > 0 && !cluster.AvailableResources.GPU.IsZero() {
            gpuClusters = append(gpuClusters, cluster)
        }
    }

    if len(gpuClusters) == 0 {
        return nil, fmt.Errorf("no GPU-capable clusters available for GPU workload")
    }

    return p.generateDistributedDistribution(analysis, gpuClusters)
}

func (p *PlacementAnalyzer) generateLatencyOptimizedDistribution(analysis *JobAnalysis, clusters []ClusterProfile) (map[string]ResourceAllocation, error) {
    // Filter clusters with low latency
    lowLatencyClusters := make([]ClusterProfile, 0)
    for _, cluster := range clusters {
        if cluster.NetworkLatency <= analysis.NetworkRequirements.Latency {
            lowLatencyClusters = append(lowLatencyClusters, cluster)
        }
    }

    if len(lowLatencyClusters) == 0 {
        return nil, fmt.Errorf("no clusters meet latency requirements")
    }

    // Prefer consolidation for latency-sensitive workloads
    return p.generateConsolidatedDistribution(analysis, lowLatencyClusters)
}

func (p *PlacementAnalyzer) generateHybridDistribution(analysis *JobAnalysis, clusters []ClusterProfile) (map[string]ResourceAllocation, error) {
    distribution := make(map[string]ResourceAllocation)

    // Place masters on high-performance clusters
    // Place workers distributed across available clusters
    
    masterTasks := make([]TaskAnalysis, 0)
    workerTasks := make([]TaskAnalysis, 0)

    for _, task := range analysis.Tasks {
        if task.TaskType == TaskTypeMaster || task.TaskType == TaskTypeCoordinator {
            masterTasks = append(masterTasks, task)
        } else {
            workerTasks = append(workerTasks, task)
        }
    }

    // Place masters on best clusters
    for i, task := range masterTasks {
        if i < len(clusters) {
            cluster := clusters[i]
            allocation := ResourceAllocation{
                ClusterName: cluster.Name,
                Tasks: []TaskAllocation{{
                    TaskName: task.Name,
                    Replicas: task.Replicas,
                    Weight:   1.0,
                }},
                TotalWeight: 1.0,
            }
            distribution[cluster.Name] = allocation
        }
    }

    // Distribute workers across remaining clusters
    availableClusters := clusters[len(masterTasks):]
    if len(availableClusters) > 0 {
        workerDistribution, err := p.generateDistributedDistribution(&JobAnalysis{Tasks: workerTasks}, availableClusters)
        if err != nil {
            return nil, err
        }

        // Merge worker distribution
        for clusterName, allocation := range workerDistribution {
            if existing, ok := distribution[clusterName]; ok {
                existing.Tasks = append(existing.Tasks, allocation.Tasks...)
                distribution[clusterName] = existing
            } else {
                distribution[clusterName] = allocation
            }
        }
    }

    return distribution, nil
}

// analyzeNetworkTopology creates network topology requirements
func (p *PlacementAnalyzer) analyzeNetworkTopology(analysis *JobAnalysis) NetworkTopology {
    topology := NetworkTopology{
        RequiredBandwidth:  make(map[string]resource.Quantity),
        LatencyMatrix:      make(map[string]map[string]time.Duration),
        CommunicationPairs: make([]CommunicationPair, 0),
    }

    // Create communication pairs based on job type
    switch analysis.JobType {
    case JobTypePyTorch, JobTypeTensorFlow:
        // All-reduce pattern: workers communicate with each other
        workers := make([]string, 0)
        for _, task := range analysis.Tasks {
            if task.TaskType == TaskTypeWorker {
                workers = append(workers, task.Name)
            }
        }

        for i, worker1 := range workers {
            for j, worker2 := range workers {
                if i != j {
                    topology.CommunicationPairs = append(topology.CommunicationPairs, CommunicationPair{
                        Source:      worker1,
                        Destination: worker2,
                        Frequency:   "high",
                        DataVolume:  resource.MustParse("100Mi"),
                    })
                }
            }
        }

    case JobTypeSpark:
        // Driver-executor pattern
        var driver string
        executors := make([]string, 0)
        
        for _, task := range analysis.Tasks {
            if task.TaskType == TaskTypeDriver {
                driver = task.Name
            } else if task.TaskType == TaskTypeExecutor {
                executors = append(executors, task.Name)
            }
        }

        for _, executor := range executors {
            topology.CommunicationPairs = append(topology.CommunicationPairs, CommunicationPair{
                Source:      driver,
                Destination: executor,
                Frequency:   "medium",
                DataVolume:  resource.MustParse("50Mi"),
            })
        }
    }

    return topology
}

// Helper scoring functions
func (p *PlacementAnalyzer) calculateResourceScore(profile ClusterProfile, analysis *JobAnalysis) float64 {
    score := 0.0

    // CPU score
    if !analysis.TotalResources.CPU.IsZero() && !profile.AvailableResources.CPU.IsZero() {
        required := float64(analysis.TotalResources.CPU.MilliValue())
        available := float64(profile.AvailableResources.CPU.MilliValue())
        if available >= required {
            utilization := required / available
            // Optimal utilization is around 70%
            if utilization >= 0.5 && utilization <= 0.8 {
                score += 0.25
            } else {
                score += 0.15 * (1.0 - math.Abs(utilization-0.7))
            }
        }
    }

    // Memory score
    if !analysis.TotalResources.Memory.IsZero() && !profile.AvailableResources.Memory.IsZero() {
        required := float64(analysis.TotalResources.Memory.Value())
        available := float64(profile.AvailableResources.Memory.Value())
        if available >= required {
            utilization := required / available
            if utilization >= 0.5 && utilization <= 0.8 {
                score += 0.25
            } else {
                score += 0.15 * (1.0 - math.Abs(utilization-0.7))
            }
        }
    }

    // GPU score
    if !analysis.TotalResources.GPU.IsZero() {
        if !profile.AvailableResources.GPU.IsZero() {
            required, _ := analysis.TotalResources.GPU.AsInt64()
            available, _ := profile.AvailableResources.GPU.AsInt64()
            if available >= required {
                score += 0.5 // High bonus for GPU availability
            }
        }
    } else {
        score += 0.25 // No GPU required, so no penalty
    }

    return math.Min(score, 1.0)
}

func (p *PlacementAnalyzer) calculateNetworkScore(profile ClusterProfile, analysis *JobAnalysis) float64 {
    score := 0.0

    // Latency score
    if profile.NetworkLatency <= analysis.NetworkRequirements.Latency {
        score += 0.5
    } else {
        ratio := float64(analysis.NetworkRequirements.Latency) / float64(profile.NetworkLatency)
        score += 0.5 * ratio
    }

    // Bandwidth score
    requiredBandwidth := int64(1000) // Default 1 Gbps
    if analysis.JobType == JobTypeTensorFlow || analysis.JobType == JobTypePyTorch {
        requiredBandwidth = 10000 // 10 Gbps for distributed training
    }

    if profile.NetworkBandwidth >= requiredBandwidth {
        score += 0.5
    } else {
        ratio := float64(profile.NetworkBandwidth) / float64(requiredBandwidth)
        score += 0.5 * ratio
    }

    return math.Min(score, 1.0)
}

func (p *PlacementAnalyzer) calculateCostScore(profile ClusterProfile, analysis *JobAnalysis) float64 {
    // Simple cost scoring - prefer lower cost
    if profile.CostPerHour <= 1.0 {
        return 1.0
    } else if profile.CostPerHour <= 5.0 {
        return 0.8
    } else if profile.CostPerHour <= 10.0 {
        return 0.6
    } else {
        return 0.4
    }
}

func (p *PlacementAnalyzer) calculatePerformanceScore(profile ClusterProfile, analysis *JobAnalysis) float64 {
    score := 0.0

    // GPU specialization
    if !analysis.TotalResources.GPU.IsZero() && len(profile.GPUTypes) > 0 {
        score += 0.5
    }

    // Framework-specific optimizations
    for _, capability := range profile.SpecialCapabilities {
        switch capability {
        case "ml-optimized":
            if analysis.JobType == JobTypeTensorFlow || analysis.JobType == JobTypePyTorch {
                score += 0.3
            }
        case "big-data-optimized":
            if analysis.JobType == JobTypeSpark || analysis.JobType == JobTypeFlink {
                score += 0.3
            }
        case "high-memory":
            memoryValue := analysis.TotalResources.Memory.Value()
            if memoryValue > 64*1024*1024*1024 { // > 64GB
                score += 0.2
            }
        }
    }

    return math.Min(score, 1.0)
}