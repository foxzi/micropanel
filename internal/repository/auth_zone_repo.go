package repository

import (
	"micropanel/internal/database"
	"micropanel/internal/models"
)

type AuthZoneRepository struct {
	db *database.DB
}

func NewAuthZoneRepository(db *database.DB) *AuthZoneRepository {
	return &AuthZoneRepository{db: db}
}

func (r *AuthZoneRepository) Create(zone *models.AuthZone) error {
	result, err := r.db.Exec(
		`INSERT INTO auth_zones (site_id, path_prefix, realm, is_enabled)
		 VALUES (?, ?, ?, ?)`,
		zone.SiteID, zone.PathPrefix, zone.Realm, zone.IsEnabled,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	zone.ID = id
	return nil
}

func (r *AuthZoneRepository) GetByID(id int64) (*models.AuthZone, error) {
	zone := &models.AuthZone{}
	err := r.db.QueryRow(
		`SELECT id, site_id, path_prefix, realm, is_enabled
		 FROM auth_zones WHERE id = ?`,
		id,
	).Scan(&zone.ID, &zone.SiteID, &zone.PathPrefix, &zone.Realm, &zone.IsEnabled)
	if err != nil {
		return nil, err
	}
	return zone, nil
}

func (r *AuthZoneRepository) GetByIDWithUsers(id int64) (*models.AuthZone, error) {
	zone, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	users, err := r.ListUsers(id)
	if err != nil {
		return nil, err
	}
	zone.Users = users
	return zone, nil
}

func (r *AuthZoneRepository) ListBySite(siteID int64) ([]*models.AuthZone, error) {
	rows, err := r.db.Query(
		`SELECT id, site_id, path_prefix, realm, is_enabled
		 FROM auth_zones WHERE site_id = ? ORDER BY path_prefix`,
		siteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zones []*models.AuthZone
	for rows.Next() {
		zone := &models.AuthZone{}
		if err := rows.Scan(&zone.ID, &zone.SiteID, &zone.PathPrefix, &zone.Realm, &zone.IsEnabled); err != nil {
			return nil, err
		}
		zones = append(zones, zone)
	}
	return zones, nil
}

func (r *AuthZoneRepository) ListBySiteWithUsers(siteID int64) ([]*models.AuthZone, error) {
	zones, err := r.ListBySite(siteID)
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		users, err := r.ListUsers(zone.ID)
		if err != nil {
			return nil, err
		}
		zone.Users = users
	}
	return zones, nil
}

func (r *AuthZoneRepository) Update(zone *models.AuthZone) error {
	_, err := r.db.Exec(
		`UPDATE auth_zones SET path_prefix = ?, realm = ?, is_enabled = ?
		 WHERE id = ?`,
		zone.PathPrefix, zone.Realm, zone.IsEnabled, zone.ID,
	)
	return err
}

func (r *AuthZoneRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM auth_zones WHERE id = ?`, id)
	return err
}

func (r *AuthZoneRepository) DeleteBySite(siteID int64) error {
	_, err := r.db.Exec(`DELETE FROM auth_zones WHERE site_id = ?`, siteID)
	return err
}

// User management

func (r *AuthZoneRepository) CreateUser(user *models.AuthZoneUser) error {
	result, err := r.db.Exec(
		`INSERT INTO auth_zone_users (auth_zone_id, username, password_hash)
		 VALUES (?, ?, ?)`,
		user.AuthZoneID, user.Username, user.PasswordHash,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

func (r *AuthZoneRepository) GetUserByID(id int64) (*models.AuthZoneUser, error) {
	user := &models.AuthZoneUser{}
	err := r.db.QueryRow(
		`SELECT id, auth_zone_id, username, password_hash
		 FROM auth_zone_users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.AuthZoneID, &user.Username, &user.PasswordHash)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthZoneRepository) ListUsers(zoneID int64) ([]models.AuthZoneUser, error) {
	rows, err := r.db.Query(
		`SELECT id, auth_zone_id, username, password_hash
		 FROM auth_zone_users WHERE auth_zone_id = ? ORDER BY username`,
		zoneID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.AuthZoneUser
	for rows.Next() {
		var user models.AuthZoneUser
		if err := rows.Scan(&user.ID, &user.AuthZoneID, &user.Username, &user.PasswordHash); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *AuthZoneRepository) UpdateUser(user *models.AuthZoneUser) error {
	_, err := r.db.Exec(
		`UPDATE auth_zone_users SET username = ?, password_hash = ?
		 WHERE id = ?`,
		user.Username, user.PasswordHash, user.ID,
	)
	return err
}

func (r *AuthZoneRepository) DeleteUser(id int64) error {
	_, err := r.db.Exec(`DELETE FROM auth_zone_users WHERE id = ?`, id)
	return err
}
