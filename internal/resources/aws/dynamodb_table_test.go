package aws_test

import (
	"testing"

	resources "github.com/infracost/infracost/internal/resources/aws"
)

func stubDescribeTable(stub *stubbedAWS) {
	stub.WhenBody(`{"TableName":""}`).Then(200, `{
    "Table": {
        "AttributeDefinitions": [],
        "TableName": "stubbed",
        "KeySchema": [],
        "TableStatus": "ACTIVE",
        "CreationDateTime": 0,
        "ProvisionedThroughput": {
            "NumberOfDecreasesToday": 0,
            "ReadCapacityUnits": 0,
            "WriteCapacityUnits": 0
        },
        "TableSizeBytes": 10737418240,
        "ItemCount": 1000,
        "TableArn": "arn:aws:dynamodb:us-east-2:012345678901:table/foo",
        "TableId": "00000000-0000-0000-0000-000000000000"
    }
	}`)
}

func TestDynamoDBStorage(t *testing.T) {
	t.SkipNow()
	stub := stubAWS(t)
	defer stub.Close()
	stubDescribeTable(stub)

	resource := resources.NewDynamoDBTable(&resources.DynamoDbTableArguments{})
	err := resource.EstimateUsage(stub.ctx, stub.usage)

	stub.expectSuccess("DynamoDB EstimateUsage", err)
	stub.expectUsage("storage_gb", int64(10))
}

func TestDynamoDBPayPerRequest(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()
	stubDescribeTable(stub)
	stub.WhenBody("MetricName=ConsumedReadCapacityUnits", "Statistics.member.1=Sum", "Unit=Count").Then(200, `
	<GetMetricStatisticsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
	  <GetMetricStatisticsResult>
			<Label>ConsumedReadCapacityUnits</Label>
			<Datapoints>
	      <member>
	        <Unit>Count</Unit>
	        <Sum>122.6</Sum>
	        <Timestamp>1970-01-01T00:00:00Z</Timestamp>
	      </member>
	    </Datapoints>
	  </GetMetricStatisticsResult>
	</GetMetricStatisticsResponse>`)
	stub.WhenBody("MetricName=ConsumedWriteCapacityUnits", "Statistics.member.1=Sum", "Unit=Count").Then(200, `
	<GetMetricStatisticsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
	  <GetMetricStatisticsResult>
			<Label>ConsumedWriteCapacityUnits</Label>
			<Datapoints>
	      <member>
	        <Unit>Count</Unit>
	        <Sum>455.9</Sum>
	        <Timestamp>1970-01-01T00:00:00Z</Timestamp>
	      </member>
	    </Datapoints>
	  </GetMetricStatisticsResult>
	</GetMetricStatisticsResponse>`)

	resource := resources.NewDynamoDBTable(&resources.DynamoDbTableArguments{
		BillingMode: "PAY_PER_REQUEST",
	})
	err := resource.EstimateUsage(stub.ctx, stub.usage)

	stub.expectSuccess("DynamoDB EstimateUsage", err)
	stub.expectUsage("monthly_read_request_units", int64(123))
	stub.expectUsage("monthly_write_request_units", int64(456))
}

func TestDynamoDBProvisioned(t *testing.T) {
	t.SkipNow()
	stub := stubAWS(t)
	defer stub.Close()
	stubDescribeTable(stub)
	resource := resources.NewDynamoDBTable(&resources.DynamoDbTableArguments{
		BillingMode: "PROVISIONED",
	})
	err := resource.EstimateUsage(stub.ctx, stub.usage)

	stub.expectSuccess("DynamoDB EstimateUsage", err)
	stub.expectUsage("monthly_read_request_units", nil)
	stub.expectUsage("monthly_write_request_units", nil)
}
