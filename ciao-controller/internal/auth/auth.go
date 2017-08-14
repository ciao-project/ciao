// Copyright (c) 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/01org/ciao/ciao-controller/internal/datastore"
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/service"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// Auth provides ciao authentication service
type Auth struct {
	ds *datastore.Datastore
}

func (auth *Auth) setupAdminIfMissing(pwhash string) error {
	if pwhash == "" {
		return nil
	}

	_, err := auth.ds.GetUserInfo("admin")
	if err == types.ErrUserNotFound {
		err := auth.ds.AddUser("admin", pwhash)
		if err != nil {
			return errors.Wrap(err, "Error adding user")
		}
	} else if err != nil {
		return err
	}

	return nil
}

// Init initialises the authentication manager
func (auth *Auth) Init(ds *datastore.Datastore, initalAdminPasswordHash string) error {
	auth.ds = ds

	err := auth.setupAdminIfMissing(initalAdminPasswordHash)
	if err != nil {
		return err
	}

	return nil
}

// Authenticate returns true if user has acess to the desired resource
func (auth *Auth) Authenticate(username, password, tenantID string) (bool, bool) {
	ui, err := auth.ds.GetUserInfo(username)
	if err != nil {
		return false, false
	}

	err = bcrypt.CompareHashAndPassword([]byte(ui.PasswordHash), []byte(password))
	if err != nil {
		return false, false
	}

	// Let admin work on any tenant
	if username == "admin" {
		return true, true
	}

	// Some API calls do not require a tenant
	if tenantID == "" {
		return true, false
	}

	// Otherwise check user has access to desired tenant
	for _, g := range ui.Grants {
		if g == tenantID {
			return true, false
		}
	}

	return false, false
}

type authHandler struct {
	Auth        *Auth
	RealHandler http.Handler
	NextHandler http.Handler
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	if !ok {
		h.NextHandler.ServeHTTP(w, r)
		return
	}

	vars := mux.Vars(r)
	if ok, isAdmin := h.Auth.Authenticate(username, password, vars["tenant"]); ok {
		ctx := service.SetPrivilege(r.Context(), isAdmin)
		ctx = service.SetUsername(ctx, username)

		h.RealHandler.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	h.NextHandler.ServeHTTP(w, r)
}

// GetHandler returns a handler for authenticating incoming requests
func (auth *Auth) GetHandler(realHandler http.Handler, nextHandler http.Handler) http.Handler {
	h := &authHandler{Auth: auth,
		RealHandler: realHandler,
		NextHandler: nextHandler,
	}
	return h
}
