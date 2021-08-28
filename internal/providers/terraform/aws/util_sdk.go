package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	log "github.com/sirupsen/logrus"
)

const timeMonth = time.Hour * 24 * 30

func sdkWarn(service string, usageType string, id string, err interface{}) {
	// HACK: too busy to figure out how to make logrus print to screen
	fmt.Printf("Error estimating %s %s usage for %s: %s\n", service, usageType, id, err)
	log.Warnf("Error estimating %s %s usage for %s: %s", service, usageType, id, err)
}

func sdkNewConfig(region string) (aws.Config, error) {
	return config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
}

func sdkNewCloudWatchClient(region string) (*cloudwatch.Client, error) {
	config, err := sdkNewConfig(region)
	if err != nil {
		return nil, err
	}
	return cloudwatch.NewFromConfig(config), nil
}

func sdkNewS3Client(region string) (*s3.Client, error) {
	config, err := sdkNewConfig(region)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(config), nil
}

// Find a filter that _doesn't_ filter by any prefix, tag, etc.
// The "unfiltered" filter is what we need to query whole-bucket request metrics.
func sdkS3FindMetricsFilter(region string, bucket string) string {
	client, err := sdkNewS3Client(region)
	if err != nil {
		sdkWarn("S3", "requests", bucket, err)
		return ""
	}
	result, err := client.ListBucketMetricsConfigurations(context.TODO(), &s3.ListBucketMetricsConfigurationsInput{
		Bucket: strPtr(bucket),
	})
	if err != nil {
		sdkWarn("S3", "requests", bucket, err)
		return ""
	}
	for _, config := range result.MetricsConfigurationList {
		if config.Filter == nil {
			return *config.Id
		}
	}
	return ""
}

// Get monthly-snapshot statistic of some metric for a bucket & one specified dimension.
func sdkGetS3MonthlyStatistics(region string, bucket string, metricName string, dimName string, dimValue string, statistic types.Statistic, unit types.StandardUnit) (*cloudwatch.GetMetricStatisticsOutput, error) {
	client, err := sdkNewCloudWatchClient(region)
	if err != nil {
		return nil, err
	}
	return client.GetMetricStatistics(context.TODO(), &cloudwatch.GetMetricStatisticsInput{
		Namespace:  strPtr("AWS/S3"),
		MetricName: strPtr(metricName),
		StartTime:  aws.Time(time.Now().Add(-timeMonth)),
		EndTime:    aws.Time(time.Now()),
		Period:     int32Ptr(60 * 60 * 24 * 30),
		Statistics: []types.Statistic{statistic},
		Unit:       unit,
		Dimensions: []types.Dimension{
			{Name: strPtr("BucketName"), Value: strPtr(bucket)},
			{Name: strPtr(dimName), Value: strPtr(dimValue)},
		},
	})
}

func sdkS3GetBucketSizeBytes(region string, bucket string, storageType string) float64 {
	stats, err := sdkGetS3MonthlyStatistics(region, bucket, "BucketSizeBytes", "StorageType", storageType, types.StatisticAverage, types.StandardUnitBytes)
	if err != nil {
		sdkWarn("S3", storageType, bucket, err)
		return 0
	} else if len(stats.Datapoints) == 0 {
		// not every bucket uses glacier, etc
		return 0
	}
	return *stats.Datapoints[0].Average
}

func sdkS3GetBucketRequests(region string, bucket string, filterName string, metrics []string) float64 {
	count := float64(0)
	for _, metric := range metrics {
		stats, err := sdkGetS3MonthlyStatistics(region, bucket, metric, "FilterId", filterName, types.StatisticSum, types.StandardUnitCount)
		if err != nil {
			desc := fmt.Sprintf("%s per filter %s", metric, filterName)
			sdkWarn("S3", desc, bucket, err)
		} else if len(stats.Datapoints) > 0 {
			count += *stats.Datapoints[0].Sum
		}
	}
	return count
}
