package backend

import (
	"fmt"
	"os"

	"go.podman.io/common/pkg/report"

	v1 "github.com/A2va/lsw/pkg/backend/v1"
	v2 "github.com/A2va/lsw/pkg/backend/v2"
	"github.com/A2va/lsw/pkg/config"
)

func Ps(noHeading bool, all bool) error {
	var bottleStatus []config.BottleStatus

	for _, bottle := range config.Get().Bottles {
		var st []config.BottleStatus
		var err error

		if bottle.Version == "v1" {
			st, err = v1.GetStatus(bottle, all)
			if err != nil {
				return err
			}
		} else if bottle.Version == "v2" {
			st, err = v2.GetStatus(bottle, all)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("not a valid backend")
		}

		bottleStatus = append(bottleStatus, st...)
	}

	headers := report.Headers(config.BottleStatus{}, map[string]string{})
	rpt := report.New(os.Stdout, "ps")
	defer rpt.Flush()

	rpt, err := rpt.Parse(report.OriginUnknown, "table {{.Name}} {{.Running}} {{.EnteredFrom}}")
	if err != nil {
		return err
	}

	if (rpt.RenderHeaders) && !noHeading {
		if err := rpt.Execute(headers); err != nil {
			return fmt.Errorf("failed to write report column headers: %w", err)
		}
	}

	return rpt.Execute(bottleStatus)
}
