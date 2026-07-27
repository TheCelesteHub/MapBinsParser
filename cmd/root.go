package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/TheCelesteHub/MapBinsParser/pkg/mapbin"
	"github.com/spf13/cobra"
)

func send(v any) {
	out, _ := json.Marshal(v)
	fmt.Println(string(out))
}

func fail(err error) {
	send(mapbin.GenericResult{Success: false, Error: err.Error()})
	os.Exit(1)
}

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "CelesteMapsBinParser",
		Short: "Celeste Tracker Map Bin Parser & Renderer CLI Helper",
	}

	zipSub := &cobra.Command{
		Use:   "zip",
		Short: "Zip subcommands for map bin parsing",
	}

	root.AddCommand(NewCountCollectiblesCmd(), NewExportMapCmd())
	zipSub.AddCommand(NewCountCollectiblesCmd(), NewExportMapCmd())
	root.AddCommand(zipSub)

	return root
}

func Execute() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		fail(err)
	}
}
