package main

import (
	"fmt"
	"time"

	"github.com/infracost/infracost/internal/output"
	"github.com/infracost/infracost/internal/providers/awsce"
	"github.com/infracost/infracost/internal/schema"
	"github.com/infracost/infracost/internal/spin"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func exploreCmd() *cli.Command {
	return &cli.Command{
		Name: "explore",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "month",
				Usage:    "e.g. 2020-10",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "tags",
				Usage:    "e.g. SVC=myapp,STAGE=prod",
				Required: false,
			},
		},
		Action: func(c *cli.Context) error {
			provider := awsce.New()
			if err := provider.ProcessArgs(c); err != nil {
				usageError(c, err.Error())
			}

			resources, err := provider.LoadResources()
			if err != nil {
				return err
			}

			spinner = spin.NewSpinner("Calculating costs")

			schema.CalculateCosts(resources)

			schema.SortResources(resources)

			month, _ := time.Parse("2006-01", c.String("month"))
			bucketName := month.Format("Jan 2006")
			out, err := output.ExploreTable(resources, bucketName)

			if err != nil {
				spinner.Fail()
				return errors.Wrap(err, "Error generating output")
			}

			spinner.Success()

			fmt.Printf("\n%s\n", string(out))

			return nil
		},
	}
}
