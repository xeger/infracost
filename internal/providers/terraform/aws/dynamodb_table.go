package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/infracost/infracost/internal/resources/aws"

	"github.com/infracost/infracost/internal/schema"
)

func GetDynamoDBTableRegistryItem() *schema.RegistryItem {
	return &schema.RegistryItem{
		Name: "aws_dynamodb_table",
		Notes: []string{
			"DAX is not yet supported.",
		},
		RFunc: NewDynamoDBTable,
	}
}

func NewDynamoDBTable(d *schema.ResourceData, u *schema.UsageData) *schema.Resource {
	region := d.Get("region").String()

	billingMode := d.Get("billing_mode").String()

	var readCapacity int64
	if d.Get("read_capacity").Exists() {
		readCapacity = d.Get("read_capacity").Int()
	}

	var writeCapacity int64
	if d.Get("write_capacity").Exists() {
		writeCapacity = d.Get("write_capacity").Int()
	}

	replicaRegions := []string{}
	if d.Get("replica").Exists() {
		for _, data := range d.Get("replica").Array() {
			replicaRegions = append(replicaRegions, data.Get("region_name").String())
		}
	}

	args := &aws.DynamoDbTableArguments{
		Address:        d.Address,
		Region:         region,
		BillingMode:    billingMode,
		WriteCapacity:  writeCapacity,
		ReadCapacity:   readCapacity,
		ReplicaRegions: replicaRegions,
	}
	args.PopulateUsage(u)

	resource := aws.NewDynamoDBTable(args)
	resource.UsageEstimate = dynamoUsageEstimate(d)
	return resource
}

func dynamoUsageEstimate(d *schema.ResourceData) schema.UsageEstimateFunc {
	return func(keys []string, usage map[string]interface{}) error {
		id := d.RawValues.Get("id").String()
		region := d.RawValues.Get("region").String()
		for _, key := range keys {
			switch key {
			case "storage_gb":
				sb := sdkDynamoGetStorageBytes(region, id)
				usage[key] = sb / (1000 * 1000 * 1000)
			case "monthly_read_request_units":
				metric, err := sdkGetMonthlyStats(sdkStatsRequest{region: region, namespace: "AWS/DynamoDB", metric: "ConsumedReadCapacityUnits", dimensions: map[string]string{"TableName": id}, statistic: types.StatisticSum, unit: types.StandardUnitCount})
				if err == nil {
					if len(metric.Datapoints) > 0 {
						usage[key] = metric.Datapoints[0].Sum
					}
				} else {
					sdkWarn("DynamoDB", key, id, err)
				}
			case "monthly_write_request_units":
				metric, err := sdkGetMonthlyStats(sdkStatsRequest{region: region, namespace: "AWS/DynamoDB", metric: "ConsumedWriteCapacityUnits", dimensions: map[string]string{"TableName": id}, statistic: types.StatisticSum, unit: types.StandardUnitCount})
				if err == nil {
					if len(metric.Datapoints) > 0 {
						usage[key] = metric.Datapoints[0].Sum
					}
				} else {
					sdkWarn("DynamoDB", key, id, err)
				}

				//or: pitr_backup_storage_gb on_demand_backup_storage_gb monthly_data_restored_gb monthly_streams_read_request_units
			}
		}
		return nil
	}
}
