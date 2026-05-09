package model

// migrateDBV2 keeps the legacy migrateDB flow intact while avoiding the
// duplicate Channel AutoMigrate call that breaks SQLite upgrades once
// model_id_prefix already exists.
func migrateDBV2() error {
	models := []interface{}{
		&Channel{},
		&ChannelKey{},
		&Token{},
		&User{},
		&Option{},
		&Redemption{},
		&Ability{},
		&Log{},
	}

	for _, currentModel := range models {
		if err := DB.AutoMigrate(currentModel); err != nil {
			return err
		}
	}

	if err := MigrateChannelKeysV2(); err != nil {
		return err
	}

	return nil
}
