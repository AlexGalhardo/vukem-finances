// Package seeders fills the database with demo data: vukem db:seed.
package seeders

import vukem "github.com/AlexGalhardo/vukem"

// All lists the seeders run by vukem db:seed, in order; make:seeder appends here.
func All() []vukem.Seeder {
	return []vukem.Seeder{}
}

// AllApp lists seeders that need the booted app. vukem db:seed runs them
// after All. Both finance seeders refuse production.
func AllApp() []vukem.AppSeeder {
	return []vukem.AppSeeder{
		FinanceAdminSeeder{},
		FinanceDemoSeeder{},
	}
}
