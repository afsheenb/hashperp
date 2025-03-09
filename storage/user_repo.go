// Append to existing storage/user_repo.go

// GetPublicKey implements UserRepository.GetPublicKey
func (r *PostgresUserRepository) GetPublicKey(ctx context.Context, userID string) ([]byte, error) {
	var dbUser DBUser
	result := r.db.WithContext(ctx).Where("id = ?", userID).First(&dbUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", result.Error)
	}
	
	return dbUser.PublicKey, nil
}
