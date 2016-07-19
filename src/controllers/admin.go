// This source file is part of the Packet Guardian project.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package controllers

import (
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/usi-lfkeitel/packet-guardian/src/common"
	"github.com/usi-lfkeitel/packet-guardian/src/models"
	"github.com/usi-lfkeitel/packet-guardian/src/reports"
	"github.com/usi-lfkeitel/packet-guardian/src/stats"
)

var (
	ipStartRegex  = regexp.MustCompile(`^[0-9]{1,3}\.`)
	macStartRegex = regexp.MustCompile(`^[0-f]{2}\:`)
)

type Admin struct {
	e *common.Environment
}

func NewAdminController(e *common.Environment) *Admin {
	return &Admin{e: e}
}

func (a *Admin) redirectToRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (a *Admin) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	sessionUser := models.GetUserFromContext(r)
	if !sessionUser.Can(models.ViewAdminPage) {
		a.redirectToRoot(w, r)
		return
	}

	deviceTotal, deviceAvg := stats.GetDeviceStats(a.e)
	data := map[string]interface{}{
		"canViewUsers": sessionUser.Can(models.ViewUsers),
		"leaseStats":   stats.GetLeaseStats(a.e),
		"deviceTotal":  deviceTotal,
		"deviceAvg":    deviceAvg,
	}
	a.e.Views.NewView("admin-dash", r).Render(w, data)
}

func (a *Admin) ManageHandler(w http.ResponseWriter, r *http.Request) {
	sessionUser := models.GetUserFromContext(r)
	if !sessionUser.Can(models.ViewDevices) {
		a.redirectToRoot(w, r)
		return
	}

	user, err := models.GetUserByUsername(a.e, mux.Vars(r)["username"])

	results, err := models.GetDevicesForUser(a.e, user)
	if err != nil {
		a.e.Log.Errorf("Error getting devices for user %s: %s", user.Username, err.Error())
		a.e.Views.RenderError(w, r, nil)
		return
	}

	data := map[string]interface{}{
		"user":               user,
		"devices":            results,
		"canCreateDevice":    sessionUser.Can(models.CreateDevice),
		"canEditDevice":      sessionUser.Can(models.EditDevice),
		"canDeleteDevice":    sessionUser.Can(models.DeleteDevice),
		"canReassignDevice":  sessionUser.Can(models.ReassignDevice),
		"canManageBlacklist": sessionUser.Can(models.ManageBlacklist),
	}

	a.e.Views.NewView("admin-manage", r).Render(w, data)
}

func (a *Admin) ShowDeviceHandler(w http.ResponseWriter, r *http.Request) {
	sessionUser := models.GetUserFromContext(r)
	if !sessionUser.Can(models.ViewDevices) {
		a.redirectToRoot(w, r)
		return
	}

	mac, err := net.ParseMAC(mux.Vars(r)["mac"])
	if err != nil {
		a.e.Views.RenderError(w, r, map[string]interface{}{
			"title": "No device found",
			"body":  "Incorrectly formed MAC address: " + mux.Vars(r)["mac"],
		})
		return
	}
	device, err := models.GetDeviceByMAC(a.e, mac)
	if err != nil {
		a.e.Log.Errorf("Error showing device %s", err.Error())
		a.e.Views.RenderError(w, r, nil)
		return
	}
	user, err := models.GetUserByUsername(a.e, device.Username)
	if err != nil {
		a.e.Log.Errorf("Error getting user %s", err.Error())
		a.e.Views.RenderError(w, r, nil)
		return
	}

	data := map[string]interface{}{
		"user":               user,
		"device":             device,
		"canEditDevice":      sessionUser.Can(models.EditDevice),
		"canDeleteDevice":    sessionUser.Can(models.DeleteDevice),
		"canReassignDevice":  sessionUser.Can(models.ReassignDevice),
		"canManageBlacklist": sessionUser.Can(models.ManageBlacklist),
	}

	a.e.Views.NewView("admin-manage-device", r).Render(w, data)
}

func (a *Admin) SearchHandler(w http.ResponseWriter, r *http.Request) {
	sessionUser := models.GetUserFromContext(r)
	if !sessionUser.Can(models.ViewAdminPage | models.ViewDevices) {
		a.redirectToRoot(w, r)
		return
	}

	query := r.FormValue("q")
	var results []*models.Device
	var err error

	if query != "" {
		if macStartRegex.MatchString(query) {
			results, err = models.SearchDevicesByField(a.e, "mac", query+"%")
		} else if ipStartRegex.MatchString(query) {
			if ip := net.ParseIP(query); ip != nil {
				// Get device with exact lease IP
				lease, err := models.GetLeaseByIP(a.e, ip)
				if err == nil && !lease.IsExpired() {
					results, err = models.SearchDevicesByField(a.e, "mac", lease.MAC.String())
				}
			} else {
				// Get leases matching partial IP
				var leases []*models.Lease
				leases, err = models.SearchLeases(
					a.e,
					`"ip" LIKE ?`,
					query+"%",
				)
				// Get devices corresponding to each lease
				var d *models.Device
				for _, l := range leases {
					if l.IsExpired() {
						continue
					}
					d, err = models.GetDeviceByMAC(a.e, l.MAC)
					if err != nil || d.ID == 0 {
						continue
					}
					results = append(results, d)
				}
			}
		} else {
			results, err = models.SearchDevicesByField(a.e, "username", query+"%")
			if len(results) == 0 {
				results, err = models.SearchDevicesByField(a.e, "user_agent", "%"+query+"%")
			}
		}
	}

	if err != nil {
		a.e.Log.Errorf("Error getting search results: %s", err.Error())
	}

	data := map[string]interface{}{
		"query":   query,
		"devices": results,
	}

	a.e.Views.NewView("admin-search", r).Render(w, data)
}

func (a *Admin) AdminUserListHandler(w http.ResponseWriter, r *http.Request) {
	sessionUser := models.GetUserFromContext(r)
	if !sessionUser.Can(models.ViewUsers) {
		a.redirectToRoot(w, r)
		return
	}

	users, err := models.GetAllUsers(a.e)
	if err != nil {
		a.e.Log.Errorf("Error getting all users: %s", err.Error())
	}

	data := map[string]interface{}{
		"users":         users,
		"canEditUser":   sessionUser.Can(models.EditUser),
		"canCreateUser": sessionUser.Can(models.CreateUser),
	}

	a.e.Views.NewView("admin-user-list", r).Render(w, data)
}

func (a *Admin) AdminUserHandler(w http.ResponseWriter, r *http.Request) {
	sessionUser := models.GetUserFromContext(r)
	if !sessionUser.Can(models.EditUser) {
		a.redirectToRoot(w, r)
		return
	}

	username := mux.Vars(r)["username"]
	user, err := models.GetUserByUsername(a.e, username)
	if err != nil {
		a.e.Log.Errorf("Error getting user %s: %s", username, err.Error())
	}

	data := map[string]interface{}{
		"user": user,
	}

	a.e.Views.NewView("admin-user", r).Render(w, data)
}

func (a *Admin) AdminLeaseListHandler(w http.ResponseWriter, r *http.Request) {
	sessionUser := models.GetUserFromContext(r)
	if !sessionUser.Can(models.ViewLeases) {
		a.redirectToRoot(w, r)
		return
	}

	network := strings.ToLower(mux.Vars(r)["network"])
	_, registered := r.URL.Query()["registered"]

	leases, err := models.SearchLeases(a.e,
		"network = ? AND registered = ? AND end > ?",
		network, registered, time.Now().Unix(),
	)
	if err != nil {
		a.e.Log.WithField("Err", err).Error("Failed to get leases")
	}

	sort.Sort(models.LeaseSorter(leases))

	data := map[string]interface{}{
		"network":    network,
		"registered": registered,
		"leases":     leases,
	}

	a.e.Views.NewView("admin-leases", r).Render(w, data)
}

func (a *Admin) ReportHandler(w http.ResponseWriter, r *http.Request) {
	sessionUser := models.GetUserFromContext(r)
	if !sessionUser.Can(models.ViewAdminPage) {
		a.redirectToRoot(w, r)
		return
	}

	report := mux.Vars(r)["report"]
	if report != "" {
		reports.RenderReport(report, w, r)
		return
	}

	allReports := reports.GetReports()
	i := 0
	reportSlice := make([]*reports.Report, len(allReports))
	for _, v := range allReports {
		reportSlice[i] = v
		i++
	}

	sort.Sort(reports.ReportSorter(reportSlice))

	data := map[string]interface{}{
		"reports": reportSlice,
	}

	a.e.Views.NewView("admin-reports", r).Render(w, data)
}
