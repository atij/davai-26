package cmd

import (
	"fmt"
	"os"

	"github.com/adoreme/geo-tracker/internal/db"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var promptsCmd = &cobra.Command{
	Use:   "prompts",
	Short: "Manage prompt library",
}

var listPromptsCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all active prompts",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Connect(cfg.Database)
		if err != nil {
			return err
		}
		defer database.Close()

		repo := db.NewPromptRepo(database)
		prompts, err := repo.ListActive()
		if err != nil {
			return err
		}

		fmt.Printf("%-4s | %-12s | %-50s\n", "ID", "Category", "Text")
		fmt.Println("-------------------------------------------------------------------------")
		for _, p := range prompts {
			text := p.Text
			if len(text) > 50 {
				text = text[:47] + "..."
			}
			fmt.Printf("%-4d | %-12s | %-50s\n", p.ID, p.Category, text)
		}
		return nil
	},
}

type PromptFile struct {
	Prompts []struct {
		Text     string `yaml:"text"`
		Category string `yaml:"category"`
		Notes    string `yaml:"notes"`
	} `yaml:"prompts"`
}

var importPromptsCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Bulk import from YAML file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}

		var pf PromptFile
		if err := yaml.Unmarshal(data, &pf); err != nil {
			return err
		}

		database, err := db.Connect(cfg.Database)
		if err != nil {
			return err
		}
		defer database.Close()

		repo := db.NewPromptRepo(database)
		var dbPrompts []db.Prompt
		for _, p := range pf.Prompts {
			dbPrompts = append(dbPrompts, db.Prompt{
				Text:     p.Text,
				Category: p.Category,
				Active:   true,
				Notes:    p.Notes,
			})
		}

		if err := repo.BulkInsert(dbPrompts); err != nil {
			return err
		}

		fmt.Printf("Successfully imported %d prompts\n", len(dbPrompts))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(promptsCmd)
	promptsCmd.AddCommand(listPromptsCmd)
	promptsCmd.AddCommand(importPromptsCmd)
}
