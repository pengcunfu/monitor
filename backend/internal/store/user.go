package store

import (
	"errors"

	"gorm.io/gorm"

	"monitor/internal/model"
)

// FindUserByUsername 按用户名查询用户。
func (s *Store) FindUserByUsername(name string) (*model.User, error) {
	var u model.User
	err := s.db.Where("username = ?", name).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindUserByID 按 ID 查询用户。
func (s *Store) FindUserByID(id uint) (*model.User, error) {
	var u model.User
	err := s.db.First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUserPassword 更新用户密码。
func (s *Store) UpdateUserPassword(id uint, hash string) error {
	return s.db.Model(&model.User{}).Where("id = ?", id).
		Update("password_hash", hash).Error
}
