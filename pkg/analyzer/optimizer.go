package analyzer

import (
    "math"
)

// Optimizer generates optimization recommendations
type Optimizer struct{}

// NewOptimizer creates a new optimizer instance
func NewOptimizer() *Optimizer {
    return &Optimizer{}
}

// GenerateRecommendations analyzes job and placement to provide optimization recommendations
func (o *Optimizer) GenerateRecommendations(analysis *JobAnalysis, placement *PlacementStrategy) *OptimizationResult {
    result := &OptimizationResult{
        Recommendations: make([]OptimizationRecommendation, 0),
    }

    // Calculate baseline costs and performance
    result.OriginalCost = o.calculateBaseCost(analysis)
    
    // Generate resource optimization recommendations
    resourceRecs := o.generateResourceRecommendations(analysis)
    result.Recommendations = append(result.Recommendations, resourceRecs...)

    // Generate network optimization recommendations
    networkRecs := o.generateNetworkRecommendations(analysis)
    result.Recommendations = append(result.Recommendations, networkRecs...)

    // Generate scheduling optimization recommendations
    schedulingRecs := o.generateSchedulingRecommendations(analysis, placement)
    result.Recommendations = append(result.Recommendations, schedulingRecs...)

    // Generate security recommendations
    securityRecs := o.generateSecurityRecommendations(analysis)
    result.Recommendations = append(result.Recommendations, securityRecs...)

    // Generate scaling recommendations
    scalingRecs := o.generateScalingRecommendations(analysis)
    result.Recommendations = append(result.Recommendations, scalingRecs...)

    // Calculate potential improvements
    result.OptimizedCost, result.CostSavings, result.PerformanceGain = o.calculateOptimizations(analysis, result.Recommendations)

    return result
}

func (o *Optimizer) calculateBaseCost(analysis *JobAnalysis) float64 {
    // Simple cost calculation based on resource requirements
    cpuCost := float64(analysis.TotalResources.CPU.MilliValue()) / 1000.0 * 0.05 // $0.05 per CPU hour
    memoryCost := float64(analysis.TotalResources.Memory.Value()) / (1024*1024*1024) * 0.01 // $0.01 per GB hour
    
    gpuCost := 0.0
    if !analysis.TotalResources.GPU.IsZero() {
        gpuValue, _ := analysis.TotalResources.GPU.AsInt64()
        gpuCost = float64(gpuValue) * 1.0 // $1.0 per GPU hour
    }

    // Multiply by estimated execution time in hours
    hours := analysis.EstimatedExecutionTime.Hours()
    return (cpuCost + memoryCost + gpuCost) * hours
}

func (o *Optimizer) generateResourceRecommendations(analysis *JobAnalysis) []OptimizationRecommendation {
    recommendations := make([]OptimizationRecommendation, 0)

    // Check for resource efficiency issues
    for _, task := range analysis.Tasks {
        if task.ResourceEfficiency < 0.7 {
            recommendations = append(recommendations, OptimizationRecommendation{
                Type:        RecommendationTypeResourceOptimization,
                Priority:    PriorityMedium,
                Description: fmt.Sprintf("Task '%s' has low resource efficiency (%.1f%%). Consider optimizing resource requests.", task.Name, task.ResourceEfficiency*100),
                Impact:      "10-20% cost reduction",
                Effort:      "Medium",
            })
        }
    }

    // Check for over-provisioned resources
    if analysis.ResourceUtilization.CPUEfficiency < 0.7 {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeResourceOptimization,
            Priority:    PriorityHigh,
            Description: "CPU utilization is low. Consider reducing CPU requests or enabling CPU throttling.",
            Impact:      "15-25% cost reduction",
            Effort:      "Low",
        })
    }

    if analysis.ResourceUtilization.MemoryEfficiency < 0.8 {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeResourceOptimization,
            Priority:    PriorityMedium,
            Description: "Memory utilization is low. Consider reducing memory requests.",
            Impact:      "10-15% cost reduction",
            Effort:      "Low",
        })
    }

    // Check for GPU optimization opportunities
    if !analysis.TotalResources.GPU.IsZero() {
        if analysis.ResourceUtilization.GPUEfficiency < 0.8 {
            recommendations = append(recommendations, OptimizationRecommendation{
                Type:        RecommendationTypeResourceOptimization,
                Priority:    PriorityCritical,
                Description: "GPU utilization is low. Consider using mixed precision training or increasing batch size.",
                Impact:      "20-40% performance improvement",
                Effort:      "Medium",
            })
        }
    }

    return recommendations
}

func (o *Optimizer) generateNetworkRecommendations(analysis *JobAnalysis) []OptimizationRecommendation {
    recommendations := make([]OptimizationRecommendation, 0)

    // Check for high network utilization
    if analysis.ResourceUtilization.NetworkUtilization > 0.8 {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeNetworkOptimization,
            Priority:    PriorityHigh,
            Description: "High network utilization detected. Consider using gradient compression or reducing communication frequency.",
            Impact:      "10-30% performance improvement",
            Effort:      "Medium",
        })
    }

    // Check for cross-cluster communication
    if len(analysis.NetworkRequirements.RequiredPorts) > 4 {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeNetworkOptimization,
            Priority:    PriorityMedium,
            Description: "Multiple communication ports detected. Consider consolidating communication channels.",
            Impact:      "5-15% latency reduction",
            Effort:      "High",
        })
    }

    // Framework-specific recommendations
    if analysis.JobType == JobTypeTensorFlow || analysis.JobType == JobTypePyTorch {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeNetworkOptimization,
            Priority:    PriorityMedium,
            Description: "For distributed training, consider using NCCL backend for better GPU communication performance.",
            Impact:      "15-25% training speed improvement",
            Effort:      "Low",
        })
    }

    return recommendations
}

func (o *Optimizer) generateSchedulingRecommendations(analysis *JobAnalysis, placement *PlacementStrategy) []OptimizationRecommendation {
    recommendations := make([]OptimizationRecommendation, 0)

    // Check for inefficient task distribution
    if placement != nil {
        clustersUsed := len(placement.ResourceDistribution)
        if clustersUsed > int(placement.RecommendedClusters) {
            recommendations = append(recommendations, OptimizationRecommendation{
                Type:        RecommendationTypeSchedulingOptimization,
                Priority:    PriorityMedium,
                Description: "Job is distributed across more clusters than recommended. Consider consolidating to reduce coordination overhead.",
                Impact:      "5-10% performance improvement",
                Effort:      "Medium",
            })
		}

        if clustersUsed < int(placement.MinimumClusters) {
            recommendations = append(recommendations, OptimizationRecommendation{
                Type:        RecommendationTypeSchedulingOptimization,
                Priority:    PriorityHigh,
                Description: "Job requires more clusters for optimal performance. Consider distributing workload across additional clusters.",
                Impact:      "20-40% performance improvement",
                Effort:      "Low",
            })
        }
    }

    // Check for bottleneck tasks
    if analysis.ScalabilityMetrics.BottleneckTask != "" {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeSchedulingOptimization,
            Priority:    PriorityCritical,
            Description: fmt.Sprintf("Task '%s' is identified as a bottleneck. Consider optimizing or scaling this component.", analysis.ScalabilityMetrics.BottleneckTask),
            Impact:      "30-50% performance improvement",
            Effort:      "High",
        })
    }

    // Check for scaling potential
    if analysis.ScalabilityMetrics.ScalingFactor > 3.0 {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeSchedulingOptimization,
            Priority:    PriorityMedium,
            Description: "Job has high scaling potential. Consider implementing horizontal pod autoscaling.",
            Impact:      "Improved resource utilization",
            Effort:      "Medium",
        })
    }

    return recommendations
}

func (o *Optimizer) generateSecurityRecommendations(analysis *JobAnalysis) []OptimizationRecommendation {
    recommendations := make([]OptimizationRecommendation, 0)

    // Check for missing security context
    if analysis.SecurityConstraints.RequiredSecurityContext == nil {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeSecurityImprovement,
            Priority:    PriorityHigh,
            Description: "No security context specified. Consider adding security constraints for better isolation.",
            Impact:      "Improved security posture",
            Effort:      "Low",
        })
    }

    // Check for service account
    if analysis.SecurityConstraints.ServiceAccountName == "" {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeSecurityImprovement,
            Priority:    PriorityMedium,
            Description: "No service account specified. Consider using a dedicated service account with minimal permissions.",
            Impact:      "Enhanced security through RBAC",
            Effort:      "Medium",
        })
    }

    // Check for network policies
    if len(analysis.SecurityConstraints.NetworkPolicies) == 0 {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeSecurityImprovement,
            Priority:    PriorityMedium,
            Description: "No network policies defined. Consider implementing network segmentation for better security.",
            Impact:      "Reduced attack surface",
            Effort:      "Medium",
        })
    }

    // Check for secrets management
    if len(analysis.SecurityConstraints.SecretRequirements) > 0 {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeSecurityImprovement,
            Priority:    PriorityHigh,
            Description: "Secrets detected. Ensure proper secret rotation and consider using external secret management.",
            Impact:      "Enhanced credential security",
            Effort:      "High",
        })
    }

    return recommendations
}

func (o *Optimizer) generateScalingRecommendations(analysis *JobAnalysis) []OptimizationRecommendation {
    recommendations := make([]OptimizationRecommendation, 0)

    // Check for horizontal scaling opportunities
    workerTasks := 0
    for _, task := range analysis.Tasks {
        if task.TaskType == TaskTypeWorker && task.Splittable {
            workerTasks++
        }
    }

    if workerTasks > 0 {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeScalingImprovement,
            Priority:    PriorityMedium,
            Description: "Worker tasks detected that can benefit from horizontal scaling. Consider implementing dynamic scaling based on workload.",
            Impact:      "Better resource utilization and cost optimization",
            Effort:      "Medium",
        })
    }

    // Check for vertical scaling opportunities
    for _, task := range analysis.Tasks {
        if task.ResourceEfficiency > 0.95 {
            recommendations = append(recommendations, OptimizationRecommendation{
                Type:        RecommendationTypeScalingImprovement,
                Priority:    PriorityLow,
                Description: fmt.Sprintf("Task '%s' has very high resource efficiency. Consider vertical scaling for better performance.", task.Name),
                Impact:      "5-15% performance improvement",
                Effort:      "Low",
            })
        }
    }

    // Check for auto-scaling recommendations
    if analysis.EstimatedExecutionTime.Hours() > 2 {
        recommendations = append(recommendations, OptimizationRecommendation{
            Type:        RecommendationTypeScalingImprovement,
            Priority:    PriorityMedium,
            Description: "Long-running job detected. Consider implementing auto-scaling to handle varying resource needs.",
            Impact:      "Cost optimization for long-running workloads",
            Effort:      "Medium",
        })
    }

    return recommendations
}

func (o *Optimizer) calculateOptimizations(analysis *JobAnalysis, recommendations []OptimizationRecommendation) (float64, float64, float64) {
    baseCost := o.calculateBaseCost(analysis)
    potentialSavings := 0.0
    potentialPerformanceGain := 0.0

    for _, rec := range recommendations {
        switch rec.Type {
        case RecommendationTypeResourceOptimization:
            if rec.Priority == PriorityHigh || rec.Priority == PriorityCritical {
                potentialSavings += 0.2 // 20% cost savings
            } else {
                potentialSavings += 0.1 // 10% cost savings
            }
        case RecommendationTypeNetworkOptimization:
            potentialPerformanceGain += 0.15 // 15% performance gain
        case RecommendationTypeSchedulingOptimization:
            if rec.Priority == PriorityCritical {
                potentialPerformanceGain += 0.4 // 40% performance gain
            } else {
                potentialPerformanceGain += 0.1 // 10% performance gain
            }
        }
    }

    // Cap the potential improvements
    potentialSavings = math.Min(potentialSavings, 0.6) // Max 60% savings
    potentialPerformanceGain = math.Min(potentialPerformanceGain, 0.8) // Max 80% performance gain

    optimizedCost := baseCost * (1.0 - potentialSavings)
    
    return optimizedCost, potentialSavings, potentialPerformanceGain
}