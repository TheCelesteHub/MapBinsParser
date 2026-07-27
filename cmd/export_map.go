package cmd

import (
	"github.com/TheCelesteHub/MapBinsParser/pkg/mapbin"
	"github.com/spf13/cobra"
)

func NewExportMapCmd() *cobra.Command {
	var exportModPath, exportMapSid, exportOutDir, exportCelesteDir string
	var exportGridOnly bool
	c := &cobra.Command{
		Use:   "export-map",
		Short: "Export room PNGs and full map image for a map SID",
		Run: func(cmd *cobra.Command, args []string) {
			opts := mapbin.ExportMapImagesOptions{GridOnly: exportGridOnly, CelesteDir: exportCelesteDir}
			res, err := mapbin.ExportMapImages(exportModPath, exportMapSid, exportOutDir, opts)
			if err != nil {
				fail(err)
			}
			send(res)
		},
	}
	c.Flags().StringVarP(&exportModPath, "mod", "m", "", "Mod zip or folder or bin path")
	c.Flags().StringVarP(&exportMapSid, "map", "p", "", "Map SID")
	c.Flags().StringVarP(&exportOutDir, "out", "o", "", "Output directory")
	c.Flags().BoolVarP(&exportGridOnly, "grid-only", "g", false, "Force flat-color grid rendering, skip real-asset resolution")
	c.Flags().StringVarP(&exportCelesteDir, "celeste-dir", "c", "", "Celeste install root (folder containing Content/) for real tile/decal asset rendering")
	c.MarkFlagRequired("mod")
	c.MarkFlagRequired("map")
	c.MarkFlagRequired("out")
	return c
}
