package cmd

import (
	"github.com/TheCelesteHub/MapBinsParser/pkg/mapbin"
	"github.com/spf13/cobra"
)

func NewExportMapImagesCmd() *cobra.Command {
	var exportModPath, exportMapSid, exportOutDir string
	c := &cobra.Command{
		Use:   "export-map-images",
		Short: "Export room PNGs and full map image for a map SID",
		Run: func(cmd *cobra.Command, args []string) {
			res, err := mapbin.ExportMapImages(exportModPath, exportMapSid, exportOutDir)
			if err != nil {
				fail(err)
			}
			send(res)
		},
	}
	c.Flags().StringVarP(&exportModPath, "mod", "m", "", "Mod zip or folder or bin path")
	c.Flags().StringVarP(&exportMapSid, "map", "p", "", "Map SID")
	c.Flags().StringVarP(&exportOutDir, "out", "o", "", "Output directory")
	c.MarkFlagRequired("mod")
	c.MarkFlagRequired("map")
	c.MarkFlagRequired("out")
	return c
}
