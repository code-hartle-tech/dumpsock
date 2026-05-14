package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/spf13/cobra"
)

type deviceRow struct {
	UDID           string `json:"udid"`
	Name           string `json:"name"`
	ProductType    string `json:"product_type"`
	ProductVersion string `json:"product_version"`
	ConnectionType string `json:"connection_type"`
}

func newDevicesCmd(stdout, stderr io.Writer) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "devices",
		Short: "List connected iPhones / iPads",
		RunE: func(_ *cobra.Command, _ []string) error {
			list, err := ios.ListDevices()
			if err != nil {
				return fmt.Errorf("usbmuxd unreachable — is the macOS service running? %w", err)
			}

			rows := make([]deviceRow, 0, len(list.DeviceList))
			for _, e := range list.DeviceList {
				udid := e.Properties.SerialNumber
				name, ptype, pver := "", "", ""
				dev, err := ios.GetDevice(udid)
				if err == nil {
					if v, err := ios.GetValues(dev); err == nil {
						name = v.Value.DeviceName
						ptype = v.Value.ProductType
						pver = v.Value.ProductVersion
					}
				}
				rows = append(rows, deviceRow{
					UDID:           udid,
					Name:           name,
					ProductType:    ptype,
					ProductVersion: pver,
					ConnectionType: e.Properties.ConnectionType,
				})
			}

			if asJSON {
				enc := json.NewEncoder(stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(rows)
			}

			if len(rows) == 0 {
				fmt.Fprintln(stderr, "no devices connected")
				return nil
			}

			fmt.Fprintf(stdout, "%-26s  %-20s  %-13s  %-7s  %s\n",
				"UDID", "NAME", "MODEL", "iOS", "CONN")
			for _, r := range rows {
				fmt.Fprintf(stdout, "%-26s  %-20s  %-13s  %-7s  %s\n",
					trunc(r.UDID, 26), trunc(r.Name, 20),
					trunc(r.ProductType, 13), trunc(r.ProductVersion, 7),
					r.ConnectionType)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
