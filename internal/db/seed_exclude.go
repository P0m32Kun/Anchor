package db

import (
	"time"

	"github.com/P0m32Kun/Anchor/internal/exclude"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// SeedDefaultExcludedDomains populates the excluded_domains table with the
// built-in default domains if they don't already exist.
func (q *Queries) SeedDefaultExcludedDomains() error {
	now := time.Now().UTC()

	for _, domain := range exclude.DefaultDomains {
		exists, err := q.ExcludedDomainExists(domain)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		d := &models.ExcludedDomain{
			ID:        util.GenerateID(),
			Domain:    domain,
			Reason:    "built-in default",
			Builtin:   true,
			CreatedAt: now,
		}
		if err := q.CreateExcludedDomain(d); err != nil {
			return err
		}
	}

	return nil
}

// LoadCustomExcludedDomains loads user-configured excluded domains from DB into the manager.
func (q *Queries) LoadCustomExcludedDomains(mgr *exclude.Manager) error {
	domains, err := q.ListCustomExcludedDomains()
	if err != nil {
		return err
	}

	custom := make(map[string]string, len(domains))
	for _, d := range domains {
		custom[d.Domain] = d.Reason
	}
	mgr.SetCustomList(custom)
	mgr.ClearDirty()
	return nil
}

// SaveCustomExcludedDomains persists the manager's custom domains to DB.
func (q *Queries) SaveCustomExcludedDomains(mgr *exclude.Manager) error {
	if !mgr.IsDirty() {
		return nil
	}

	custom := mgr.GetCustomList()

	// Get current DB state
	dbDomains, err := q.ListCustomExcludedDomains()
	if err != nil {
		return err
	}
	dbSet := make(map[string]bool, len(dbDomains))
	for _, d := range dbDomains {
		dbSet[d.Domain] = true
	}

	now := time.Now().UTC()

	// Add new domains
	for domain, reason := range custom {
		if !dbSet[domain] {
			d := &models.ExcludedDomain{
				ID:        util.GenerateID(),
				Domain:    domain,
				Reason:    reason,
				Builtin:   false,
				CreatedAt: now,
			}
			if err := q.CreateExcludedDomain(d); err != nil {
				return err
			}
		}
	}

	// Remove deleted domains
	for _, d := range dbDomains {
		if _, ok := custom[d.Domain]; !ok {
			if err := q.DeleteExcludedDomain(d.Domain); err != nil {
				return err
			}
		}
	}

	mgr.ClearDirty()
	return nil
}
