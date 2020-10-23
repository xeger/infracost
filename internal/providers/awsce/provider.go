package awsce

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/infracost/infracost/internal/schema"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/urfave/cli/v2"
)

type awsceProvider struct {
	startTime time.Time
	endTime   time.Time
	tags      []string
}

func New() schema.Provider {
	return &awsceProvider{}
}

func (p *awsceProvider) ProcessArgs(c *cli.Context) error {
	var err error

	month := c.String("month")

	p.startTime, err = time.Parse("2006-01", month)
	if err != nil {
		return errors.New("In valid month format")
	}

	p.endTime = p.startTime.AddDate(0, 1, 0)

	p.tags = strings.Split(c.String("tags"), ",")

	return nil
}

func (p *awsceProvider) LoadResources() ([]*schema.Resource, error) {
	out, err := p.queryCostExplorer()
	if err != nil {
		return []*schema.Resource{}, errors.Wrap(err, "Error retrieving AWS cost explorer costs")
	}

	resourceMap := buildResourceMap(out)

	resources := make([]*schema.Resource, 0)
	for _, r := range resourceMap {
		resources = append(resources, r)
	}

	return resources, nil
}

func (p *awsceProvider) queryCostExplorer() (*costexplorer.GetCostAndUsageOutput, error) {
	sess := session.Must(session.NewSession())
	ce := costexplorer.New(sess)

	in := &costexplorer.GetCostAndUsageInput{
		Metrics: []*string{aws.String("UNBLENDED_COST"), aws.String("USAGE_QUANTITY")},
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(p.startTime.Format("2006-01-02")),
			End:   aws.String(p.endTime.Format("2006-01-02")),
		},
		Granularity: aws.String("MONTHLY"),
		GroupBy: []*costexplorer.GroupDefinition{
			{
				Type: aws.String("DIMENSION"),
				Key:  aws.String("SERVICE"),
			},
			{
				Type: aws.String("DIMENSION"),
				Key:  aws.String("USAGE_TYPE"),
			},
		},
		Filter: p.buildFilter(),
	}

	return ce.GetCostAndUsage(in)
}

func (p *awsceProvider) buildFilter() *costexplorer.Expression {
	tagExps := make([]*costexplorer.Expression, 0, len(p.tags))
	for _, tag := range p.tags {
		p := strings.SplitN(tag, "=", 2)

		key := aws.String(p[0])

		vals := make([]*string, 0)
		if len(p) > 1 {
			vals = []*string{aws.String(p[1])}
		}

		tagExps = append(tagExps, &costexplorer.Expression{
			Tags: &costexplorer.TagValues{
				Key:    key,
				Values: vals,
			},
		})
	}

	var filter *costexplorer.Expression
	if len(tagExps) == 1 {
		filter = tagExps[0]
	} else if len(tagExps) > 1 {
		filter = &costexplorer.Expression{
			And: tagExps,
		}
	}

	return filter
}

func buildResourceMap(out *costexplorer.GetCostAndUsageOutput) map[string]*schema.Resource {
	resourceMap := make(map[string]*schema.Resource)

	res := out.ResultsByTime[0]

	for _, group := range res.Groups {
		monthlyCost, _ := decimal.NewFromString(aws.StringValue(group.Metrics["UnblendedCost"].Amount))
		if monthlyCost.Equal(decimal.Zero) {
			continue
		}

		service := aws.StringValue(group.Keys[0])
		usageType := aws.StringValue(group.Keys[1])

		usageQuantity, _ := decimal.NewFromString(aws.StringValue(group.Metrics["UsageQuantity"].Amount))

		r, ok := resourceMap[service]
		if !ok {
			r = &schema.Resource{
				Name: service,
			}
			resourceMap[service] = r
		}

		c := &schema.CostComponent{
			Name:           usageType,
			Unit:           aws.StringValue(group.Metrics["UsageQuantity"].Unit),
			BucketQuantity: &usageQuantity,
			BucketCost:     &monthlyCost,
		}
		r.CostComponents = append(r.CostComponents, c)
	}

	return resourceMap
}
