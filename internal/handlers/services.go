package handlers

import "github.com/solaris-soft/heartcave-backend/internal/database"

type ServicesHandler struct {
	DB database.Querier
}
