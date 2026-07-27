package cmd

import (
	"github.com/TheCelesteHub/MapBinsParser/pkg/mapbin"
	"github.com/spf13/cobra"
)

func NewCountCollectiblesCmd() *cobra.Command {
	var modPath string
	c := &cobra.Command{
		Use:   "count-collectibles",
		Short: "Count map collectibles in a mod",
		Run: func(cmd *cobra.Command, args []string) {
			res, err := mapbin.CountCollectibles(modPath)
			if err != nil {
				fail(err)
			}
			send(res)
		},
	}
	c.Flags().StringVarP(&modPath, "mod", "m", "", "Mod zip or folder path")
	c.MarkFlagRequired("mod")
	return c
}
