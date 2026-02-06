package main

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"micropanel/internal/config"
	"micropanel/internal/database"
	"micropanel/internal/repository"
	"micropanel/internal/services"
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "Manage sites",
	Long:  "Create, list, and manage hosted sites.",
}

var siteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sites",
	Run:   runSiteList,
}

var siteCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new site",
	Run:   runSiteCreate,
}

var siteDeleteCmd = &cobra.Command{
	Use:   "delete [site_id]",
	Short: "Delete a site",
	Args:  cobra.ExactArgs(1),
	Run:   runSiteDelete,
}

var siteEnableCmd = &cobra.Command{
	Use:   "enable [site_id]",
	Short: "Enable a site",
	Args:  cobra.ExactArgs(1),
	Run:   runSiteEnable,
}

var siteDisableCmd = &cobra.Command{
	Use:   "disable [site_id]",
	Short: "Disable a site",
	Args:  cobra.ExactArgs(1),
	Run:   runSiteDisable,
}

var (
	siteName    string
	siteOwnerID int64
)

func init() {
	rootCmd.AddCommand(siteCmd)
	siteCmd.AddCommand(siteListCmd)
	siteCmd.AddCommand(siteCreateCmd)
	siteCmd.AddCommand(siteDeleteCmd)
	siteCmd.AddCommand(siteEnableCmd)
	siteCmd.AddCommand(siteDisableCmd)

	siteCreateCmd.Flags().StringVarP(&siteName, "name", "n", "", "Site name (required)")
	siteCreateCmd.Flags().Int64VarP(&siteOwnerID, "owner", "o", 0, "Owner user ID (required)")
	siteCreateCmd.MarkFlagRequired("name")
	siteCreateCmd.MarkFlagRequired("owner")
}

func getSiteService() (*services.SiteService, *repository.SiteRepository, func()) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	siteRepo := repository.NewSiteRepository(db)
	domainRepo := repository.NewDomainRepository(db)
	siteService := services.NewSiteService(siteRepo, domainRepo, cfg)

	return siteService, siteRepo, func() { db.Close() }
}

func runSiteList(cmd *cobra.Command, args []string) {
	_, repo, cleanup := getSiteService()
	defer cleanup()

	sites, err := repo.ListAll()
	if err != nil {
		log.Fatalf("Failed to list sites: %v", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tOWNER\tENABLED\tDOMAINS\tCREATED")
	for _, s := range sites {
		enabled := "yes"
		if !s.IsEnabled {
			enabled = "no"
		}
		domains := "-"
		if len(s.Domains) > 0 {
			domains = fmt.Sprintf("%d", len(s.Domains))
		}
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\n",
			s.ID, s.Name, s.OwnerID, enabled, domains, s.CreatedAt.Format("2006-01-02 15:04"))
	}
	w.Flush()
}

func runSiteCreate(cmd *cobra.Command, args []string) {
	svc, _, cleanup := getSiteService()
	defer cleanup()

	site, err := svc.Create(siteName, siteOwnerID)
	if err != nil {
		log.Fatalf("Failed to create site: %v", err)
	}

	fmt.Printf("Site '%s' created successfully (ID: %d)\n", siteName, site.ID)
}

func runSiteDelete(cmd *cobra.Command, args []string) {
	var siteID int64
	if _, err := fmt.Sscanf(args[0], "%d", &siteID); err != nil {
		log.Fatalf("Invalid site ID: %s", args[0])
	}

	svc, repo, cleanup := getSiteService()
	defer cleanup()

	site, err := repo.GetByID(siteID)
	if err != nil {
		log.Fatalf("Site not found: %d", siteID)
	}

	if err := svc.Delete(site.ID); err != nil {
		log.Fatalf("Failed to delete site: %v", err)
	}

	fmt.Printf("Site '%s' (ID: %d) deleted successfully\n", site.Name, siteID)
}

func runSiteEnable(cmd *cobra.Command, args []string) {
	var siteID int64
	if _, err := fmt.Sscanf(args[0], "%d", &siteID); err != nil {
		log.Fatalf("Invalid site ID: %s", args[0])
	}

	_, repo, cleanup := getSiteService()
	defer cleanup()

	site, err := repo.GetByID(siteID)
	if err != nil {
		log.Fatalf("Site not found: %d", siteID)
	}

	site.IsEnabled = true
	if err := repo.Update(site); err != nil {
		log.Fatalf("Failed to enable site: %v", err)
	}

	fmt.Printf("Site '%s' enabled\n", site.Name)
}

func runSiteDisable(cmd *cobra.Command, args []string) {
	var siteID int64
	if _, err := fmt.Sscanf(args[0], "%d", &siteID); err != nil {
		log.Fatalf("Invalid site ID: %s", args[0])
	}

	_, repo, cleanup := getSiteService()
	defer cleanup()

	site, err := repo.GetByID(siteID)
	if err != nil {
		log.Fatalf("Site not found: %d", siteID)
	}

	site.IsEnabled = false
	if err := repo.Update(site); err != nil {
		log.Fatalf("Failed to disable site: %v", err)
	}

	fmt.Printf("Site '%s' disabled\n", site.Name)
}
