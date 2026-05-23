package schema

import "github.com/downballot/downballot/internal/schema/sqltype"

// User is a user of the system.
//
// A user will be a candidate, a campaign volunteer, a campaign staffer, etc.
//
// A user must be part of an organization in order to do anything interesting.
type User struct {
	ID                uint64 `gorm:"column:id;primaryKey;not null;autoIncrement"`
	Username          string `gorm:"column:username;size:256;unique;type:varchar(256) collate nocase"`
	Name              string `gorm:"column:name;size:256;type:varchar(256) collate nocase"`
	SessionIdentifier uint64 `gorm:"column:session_identifier;not null;default:0"`
}

func (User) TableName() string {
	return "user"
}

// UserGruopMap maps a user to a group.
type UserGroupMap struct {
	ID      uint64 `gorm:"column:id;primaryKey;not null;autoIncrement"`
	UserID  uint64 `gorm:"column:user_id;not null;uniqueIndex:idx_unique_user_group,priority:1"`
	User    *User  `json:"-" gorm:"belongsTo;constraint:fk_user_group_map_user,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:user_id;references:id"`
	GroupID uint64 `gorm:"column:group_id;not null;uniqueIndex:idx_unique_user_group,priority:2"`
	Group   *Group `json:"-" gorm:"belongsTo;constraint:fk_user_group_map_group,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:group_id;references:id"`
	// TODO: Attach a role of some kind?
}

func (UserGroupMap) TableName() string {
	return "user_group_map"
}

// UserOrganizationMap maps a user to an organization.
type UserOrganizationMap struct {
	ID             uint64        `gorm:"column:id;primaryKey;not null;autoIncrement"`
	UserID         uint64        `gorm:"column:user_id;not null;uniqueIndex:idx_unique_user_organization,priority:1"`
	User           *User         `json:"-" gorm:"belongsTo;constraint:fk_user_organization_map_user,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:user_id;references:id"`
	OrganizationID uint64        `gorm:"column:organization_id;not null;uniqueIndex:idx_unique_user_organization,priority:2"`
	Organization   *Organization `json:"-" gorm:"belongsTo;constraint:fk_user_organization_map_organization,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:organization_id;references:id"`
	// TODO: Attach a role of some kind?
}

func (UserOrganizationMap) TableName() string {
	return "user_organization_map"
}

// UserTOTP is a Time-Based One-Time Password (TOTP) for a user.
//
// This is created internally by the system, and we use this to send an e-mail to the user with a password.
// The user does not have a traditional password.
type UserTOTP struct {
	ID     uint64                  `gorm:"column:id;primaryKey;not null;autoIncrement"`
	UserID uint64                  `gorm:"column:user_id;not null;uniqueIndex:idx_unique_user_totp,priority:1"`
	User   *User                   `json:"-" gorm:"belongsTo;constraint:fk_user_totp_user,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:user_id;references:id"`
	Secret sqltype.EncryptedString `gorm:"column:secret;not null;size:256;type:varchar(256) collate nocase"`
}

func (UserTOTP) TableName() string {
	return "user_totp"
}
