package main

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	"micropanel/internal/config"
	"micropanel/internal/database"
	"micropanel/internal/models"
	"micropanel/internal/repository"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	Long:  "Create, list, and manage panel users.",
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Run:   runUserList,
}

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	Run:   runUserCreate,
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete [email]",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	Run:   runUserDelete,
}

var userResetPasswordCmd = &cobra.Command{
	Use:   "reset-password [email]",
	Short: "Reset user password",
	Args:  cobra.ExactArgs(1),
	Run:   runUserResetPassword,
}

var (
	userEmail    string
	userPassword string
	userRole     string
)

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userResetPasswordCmd)

	userCreateCmd.Flags().StringVarP(&userEmail, "email", "e", "", "User email (required)")
	userCreateCmd.Flags().StringVarP(&userPassword, "password", "p", "", "User password (required)")
	userCreateCmd.Flags().StringVarP(&userRole, "role", "r", "user", "User role (admin/user)")
	userCreateCmd.MarkFlagRequired("email")
	userCreateCmd.MarkFlagRequired("password")

	userResetPasswordCmd.Flags().StringVarP(&userPassword, "password", "p", "", "New password (required)")
	userResetPasswordCmd.MarkFlagRequired("password")
}

func getUserRepo() (*repository.UserRepository, func()) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	return repository.NewUserRepository(db), func() { db.Close() }
}

func runUserList(cmd *cobra.Command, args []string) {
	repo, cleanup := getUserRepo()
	defer cleanup()

	users, err := repo.List()
	if err != nil {
		log.Fatalf("Failed to list users: %v", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tEMAIL\tROLE\tACTIVE\tCREATED")
	for _, u := range users {
		active := "yes"
		if !u.IsActive {
			active = "no"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			u.ID, u.Email, u.Role, active, u.CreatedAt.Format("2006-01-02 15:04"))
	}
	w.Flush()
}

func runUserCreate(cmd *cobra.Command, args []string) {
	repo, cleanup := getUserRepo()
	defer cleanup()

	existing, _ := repo.GetByEmail(userEmail)
	if existing != nil {
		log.Fatalf("User with email %s already exists", userEmail)
	}

	if userRole != "admin" && userRole != "user" {
		log.Fatalf("Invalid role: %s (must be admin or user)", userRole)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	user := &models.User{
		Email:        userEmail,
		PasswordHash: string(hash),
		Role:         models.Role(userRole),
		IsActive:     true,
	}

	if err := repo.Create(user); err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Printf("User %s created successfully (ID: %d, role: %s)\n", userEmail, user.ID, userRole)
}

func runUserDelete(cmd *cobra.Command, args []string) {
	email := args[0]
	repo, cleanup := getUserRepo()
	defer cleanup()

	user, err := repo.GetByEmail(email)
	if err != nil {
		log.Fatalf("User not found: %s", email)
	}

	if err := repo.Delete(user.ID); err != nil {
		log.Fatalf("Failed to delete user: %v", err)
	}

	fmt.Printf("User %s deleted successfully\n", email)
}

func runUserResetPassword(cmd *cobra.Command, args []string) {
	email := args[0]
	repo, cleanup := getUserRepo()
	defer cleanup()

	user, err := repo.GetByEmail(email)
	if err != nil {
		log.Fatalf("User not found: %s", email)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	user.PasswordHash = string(hash)
	if err := repo.Update(user); err != nil {
		log.Fatalf("Failed to update password: %v", err)
	}

	fmt.Printf("Password for %s updated successfully\n", email)
}
