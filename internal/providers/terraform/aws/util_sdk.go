package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	log "github.com/sirupsen/logrus"
)

const timeMonth = time.Hour * 24 * 30

func sdkWarn(flavor string, id string, err interface{}) {
	log.Warnf("Error estimating %s usage for %s: %s", flavor, id, err)
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

// Get monthly-snapshot statistic of some metric & dimension.
func sdkGetS3Metrics(region string, bucket string, storageType string, metricName string, statistic types.Statistic, unit types.StandardUnit) (*cloudwatch.GetMetricStatisticsOutput, error) {
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
			{Name: strPtr("StorageType"), Value: strPtr(storageType)},
		},
	})
}

func sdkGetS3BucketSizeBytes(region string, bucket string, storageType string) float64 {
	stats, err := sdkGetS3Metrics(region, bucket, storageType, "BucketSizeBytes", types.StatisticAverage, types.StandardUnitBytes)
	if err != nil {
		sdkWarn(storageType, bucket, err)
		return 0
	} else if len(stats.Datapoints) == 0 {
		// not every bucket uses glacier, etc
		return 0
	}
	return *stats.Datapoints[0].Average
}

func sdkGetS3BucketRequests(region string, bucket string, storageType string, metrics []string) float64 {
	count := float64(0)
	for _, metric := range metrics {
		stats, err := sdkGetS3Metrics(region, bucket, storageType, metric, types.StatisticAverage, types.StandardUnitBytes)
		if err != nil {
			sdkWarn(storageType, bucket, err)
			return 0
		} else if len(stats.Datapoints) > 0 {
			count += *stats.Datapoints[0].Average
		} else {
			// TODO: get this working for at least SOME cases, before suppressing this error
			sdkWarn(fmt.Sprintf("%s %s", storageType, metric), bucket, "no datapoints in metric")
		}
	}
	return count
}
