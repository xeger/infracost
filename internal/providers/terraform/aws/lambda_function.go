package aws

import (
	"github.com/infracost/infracost/internal/resources/aws"
	"github.com/infracost/infracost/internal/schema"
	"github.com/tidwall/gjson"
)

func GetLambdaFunctionRegistryItem() *schema.RegistryItem {
	return &schema.RegistryItem{
		Name:  "aws_lambda_function",
		Notes: []string{"Provisioned concurrency is not yet supported."},
		RFunc: NewLambdaFunction,
	}
}

func NewLambdaFunction(d *schema.ResourceData, u *schema.UsageData) *schema.Resource {
	region := d.Get("region").String()

	var memorySize int64 = 128
	if d.Get("memory_size").Type != gjson.Null {
		memorySize = d.Get("memory_size").Int()
	}

	args := &aws.LambdaFunctionArguments{
		Address:    d.Address,
		Region:     region,
		MemorySize: memorySize,
	}
	args.PopulateUsage(u)

	resource := aws.NewLambdaFunction(args)
	resource.UsageEstimate = lambdaUsageEstimate(d)
	return resource
}

func lambdaUsageEstimate(d *schema.ResourceData) schema.UsageEstimateFunc {
	return func(keys []string, usage map[string]interface{}) error {
		region := d.RawValues.Get("region").String()
		fn := d.RawValues.Get("id").String()

		// HACK: we disregard usage schema type & assign floats (or whatever)!!!
		for _, key := range keys {
			switch key {
			case "monthly_requests":
				usage[key] = sdkLambdaGetInvocations(region, fn)
			case "request_duration_ms":
				usage[key] = sdkLambdaGetDuration(region, fn)
			}
		}

		return nil
	}
}
