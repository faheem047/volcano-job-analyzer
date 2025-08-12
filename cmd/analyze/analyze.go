package analyze

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "strconv"
    "time"

    "volcano-job-analyzer/pkg/analyzer"
    "volcano-job-analyzer/pkg/parser"
    "volcano-job-analyzer/pkg/utils"

    "github.com/olekukonko/tablewriter"
    "go.uber.org/zap"
    "sigs.k8s.io/yaml"
)

// Execute runs the analyze command
func Execute(ctx context.Context, args []string, config *utils.Config, logger *zap.Logger, outputFormat string) error {
    if len(args) == 0 {
        return fmt.Errorf("job file path is required")
    }

    jobFile := args[0]
    logger.Info("Starting job analysis", "file", jobFile, "format", outputFormat)

    // Parse the job
    job, err := parser.ParseVolcanoJobFromFile(jobFile)
    if err != nil {
        return fmt.Errorf("failed to parse job file: %w", err)
    }

    // Create analyzer
    analyzerConfig := &analyzer.AnalyzerConfig{
        EnableAdvancedMetrics:       config.Analyzer.EnableAdvancedMetrics,
        NetworkLatencyThreshold:     config.Analyzer.NetworkLatencyThreshold,
        ResourceEfficiencyThreshold: config.Analyzer.ResourceEfficiencyThreshold,
        DefaultJobTimeout:           config.Analyzer.DefaultJobTimeout,
    }

    a := analyzer.NewAnalyzer(analyzerConfig)

    // Analyze the job
    analysis, err := a.AnalyzeJob(job)
    if err != nil {
        return fmt.Errorf("failed to analyze job: %w", err)
    }

    // Generate placement strategy if cluster profiles are available
    var placementStrategy *analyzer.PlacementStrategy
    if config.ClusterProfilesPath != "" {
        clusterProfiles, err := loadClusterProfiles(config.ClusterProfilesPath)
        if err != nil {
            logger.Warn("Failed to load cluster profiles", "error", err)
        } else {
            placementConfig := &analyzer.PlacementConfig{
                MaxClusters:                   config.Placement.MaxClusters,
                MinClusterUtilization:        config.Placement.MinClusterUtilization,
                MaxClusterUtilization:        config.Placement.MaxClusterUtilization,
                NetworkLatencyWeight:          config.Placement.NetworkLatencyWeight,
                ResourceAffinityWeight:        config.Placement.ResourceAffinityWeight,
                CostOptimizationWeight:        config.Placement.CostOptimizationWeight,
                PerformanceOptimizationWeight: config.Placement.PerformanceOptimizationWeight,
            }

            placementAnalyzer := analyzer.NewPlacementAnalyzer(placementConfig)
            placementStrategy, err = placementAnalyzer.GeneratePlacementStrategy(analysis, clusterProfiles)
            if err != nil {
                logger.Warn("Failed to generate placement strategy", "error", err)
            }
        }
    }

    // Generate optimization recommendations
    optimizer := analyzer.NewOptimizer()
    optimizationResult := optimizer.GenerateRecommendations(analysis, placementStrategy)

    // Output results
    switch outputFormat {
    case "json":
        return outputJSON(analysis, placementStrategy, optimizationResult)
    case "yaml":
        return outputYAML(analysis, placementStrategy, optimizationResult)
    case "table":
        return outputTable(analysis, placementStrategy, optimizationResult)
    default:
        return fmt.Errorf("unsupported output format: %s", outputFormat)
    }
}

func outputTable(analysis *analyzer.JobAnalysis, placement *analyzer.PlacementStrategy, optimization *analyzer.OptimizationResult) error {
    // Job Overview Table
    fmt.Println("Job Analysis:", analysis.JobName)
    fmt.Println("=" * 50)

    overviewTable := tablewriter.NewWriter(os.Stdout)
    overviewTable.SetHeader([]string{"Metric", "Value"})
    overviewTable.SetAutoWrapText(false)

    overviewTable.Append([]string{"Job Name", analysis.JobName})
    overviewTable.Append([]string{"Namespace", analysis.Namespace})
    overviewTable.Append([]string{"Job Type", analysis.JobType.String()})
    overviewTable.Append([]string{"Total Tasks", strconv.Itoa(len(analysis.Tasks))})
    overviewTable.Append([]string{"Total CPU", analysis.TotalResources.CPU.String()})
    overviewTable.Append([]string{"Total Memory", analysis.TotalResources.Memory.String()})
    overviewTable.Append([]string{"Total GPU", analysis.TotalResources.GPU.String()})
    overviewTable.Append([]string{"Estimated Duration", analysis.EstimatedExecutionTime.String()})

    fmt.Println("\nResource Requirements:")
    overviewTable.Render()

    // Task Breakdown Table
    fmt.Println("\nTask Breakdown:")
    taskTable := tablewriter.NewWriter(os.Stdout)
    taskTable.SetHeader([]string{"Task Name", "Type", "Replicas", "CPU", "Memory", "GPU", "Splittable", "Efficiency"})

    for _, task := range analysis.Tasks {
        efficiency := fmt.Sprintf("%.1f%%", task.ResourceEfficiency*100)
        taskTable.Append([]string{
            task.Name,
            task.TaskType.String(),
            strconv.Itoa(int(task.Replicas)),
            task.Resources.CPU.String(),
            task.Resources.Memory.String(),
            task.Resources.GPU.String(),
            strconv.FormatBool(task.Splittable),
            efficiency,
        })
    }
    taskTable.Render()

    // Scalability Metrics
    fmt.Println("\nScalability Metrics:")
    scalabilityTable := tablewriter.NewWriter(os.Stdout)
    scalabilityTable.SetHeader([]string{"Metric", "Value"})
    scalabilityTable.Append([]string{"Minimum Nodes", strconv.Itoa(int(analysis.ScalabilityMetrics.MinimumNodes))})
    scalabilityTable.Append([]string{"Optimal Nodes", strconv.Itoa(int(analysis.ScalabilityMetrics.OptimalNodes))})
    scalabilityTable.Append([]string{"Maximum Nodes", strconv.Itoa(int(analysis.ScalabilityMetrics.MaximumNodes))})
    scalabilityTable.Append([]string{"Scaling Factor", fmt.Sprintf("%.1fx", analysis.ScalabilityMetrics.ScalingFactor)})
    scalabilityTable.Append([]string{"Bottleneck Task", analysis.ScalabilityMetrics.BottleneckTask})
    scalabilityTable.Render()

    // Network Requirements
    fmt.Println("\nNetwork Requirements:")
    networkTable := tablewriter.NewWriter(os.Stdout)
    networkTable.SetHeader([]string{"Requirement", "Value"})
    networkTable.Append([]string{"Communication Pattern", analysis.NetworkRequirements.CommunicationPattern})
    networkTable.Append([]string{"Required Bandwidth", analysis.NetworkRequirements.Bandwidth.String()})
    networkTable.Append([]string{"Latency Requirement", analysis.NetworkRequirements.Latency.String()})
    networkTable.Append([]string{"Required Ports", fmt.Sprintf("%d ports", len(analysis.NetworkRequirements.RequiredPorts))})
    networkTable.Render()

    // Placement Strategy (if available)
    if placement != nil {
        fmt.Println("\nPlacement Strategy:", placement.Strategy.String())
        placementTable := tablewriter.NewWriter(os.Stdout)
        placementTable.SetHeader([]string{"Metric", "Value"})
        placementTable.Append([]string{"Minimum Clusters", strconv.Itoa(int(placement.MinimumClusters))})
        placementTable.Append([]string{"Recommended Clusters", strconv.Itoa(int(placement.RecommendedClusters))})
        placementTable.Append([]string{"Anti-Affinity Rules", strconv.Itoa(len(placement.AntiAffinityRules))})
        placementTable.Render()

        if len(placement.ResourceDistribution) > 0 {
            fmt.Println("\nResource Distribution:")
            distributionTable := tablewriter.NewWriter(os.Stdout)
            distributionTable.SetHeader([]string{"Cluster", "Tasks", "Total Weight"})
            
            for clusterName, allocation := range placement.ResourceDistribution {
                taskNames := make([]string, len(allocation.Tasks))
                for i, task := range allocation.Tasks {
                    taskNames[i] = fmt.Sprintf("%s(%d)", task.TaskName, task.Replicas)
                }
                distributionTable.Append([]string{
                    clusterName,
                    fmt.Sprintf("%v", taskNames),
                    fmt.Sprintf("%.1f%%", allocation.TotalWeight*100),
                })
            }
            distributionTable.Render()
        }
    }

    // Optimization Recommendations
    if optimization != nil && len(optimization.Recommendations) > 0 {
        fmt.Println("\nOptimization Recommendations:")
        
        priorityGroups := map[analyzer.Priority][]analyzer.OptimizationRecommendation{
            analyzer.PriorityCritical: {},
            analyzer.PriorityHigh:     {},
            analyzer.PriorityMedium:   {},
            analyzer.PriorityLow:      {},
        }

        for _, rec := range optimization.Recommendations {
            priorityGroups[rec.Priority] = append(priorityGroups[rec.Priority], rec)
        }

        for priority, recs := range priorityGroups {
            if len(recs) > 0 {
                fmt.Printf("\n%s Priority:\n", priority.String())
                for _, rec := range recs {
                    fmt.Printf("  [%s] %s\n", rec.Type.String(), rec.Description)
                    if rec.Impact != "" {
                        fmt.Printf("    Impact: %s\n", rec.Impact)
                    }
                }
            }
        }

        if optimization.CostSavings > 0 {
            fmt.Printf("\nPotential Cost Savings: %.1f%%\n", optimization.CostSavings*100)
        }
        if optimization.PerformanceGain > 0 {
            fmt.Printf("Potential Performance Gain: %.1f%%\n", optimization.PerformanceGain*100)
        }
    }

    return nil
}

func outputJSON(analysis *analyzer.JobAnalysis, placement *analyzer.PlacementStrategy, optimization *analyzer.OptimizationResult) error {
    output := map[string]interface{}{
        "analysis": analysis,
    }

    if placement != nil {
        output["placement"] = placement
    }

    if optimization != nil {
        output["optimization"] = optimization
    }

    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(output)
}

func outputYAML(analysis *analyzer.JobAnalysis, placement *analyzer.PlacementStrategy, optimization *analyzer.OptimizationResult) error {
    output := map[string]interface{}{
        "analysis": analysis,
    }

    if placement != nil {
        output["placement"] = placement
    }

    if optimization != nil {
        output["optimization"] = optimization
    }

    data, err := yaml.Marshal(output)
    if err != nil {
        return fmt.Errorf("failed to marshal YAML: %w", err)
    }

    fmt.Print(string(data))
    return nil
}

func loadClusterProfiles(path string) ([]analyzer.ClusterProfile, error) {
    // Implementation to load cluster profiles from file
    // This would typically read from a YAML or JSON file
    return []analyzer.ClusterProfile{
        {
            Name: "gpu-cluster-us-east",
            AvailableResources: analyzer.ResourceRequirements{
                // CPU, Memory, GPU would be populated from file
            },
            NetworkBandwidth: 10000, // 10 Gbps
            NetworkLatency:   5 * time.Millisecond,
            CostPerHour:     2.5,
            GPUTypes:        []string{"nvidia-tesla-v100"},
            Zone:            "us-east-1a",
        },
        {
            Name: "gpu-cluster-us-west",
            AvailableResources: analyzer.ResourceRequirements{
                // CPU, Memory, GPU would be populated from file
            },
            NetworkBandwidth: 10000, // 10 Gbps
            NetworkLatency:   8 * time.Millisecond,
            CostPerHour:     2.8,
            GPUTypes:        []string{"nvidia-tesla-v100"},
            Zone:            "us-west-2a",
        },
    }, nil
}