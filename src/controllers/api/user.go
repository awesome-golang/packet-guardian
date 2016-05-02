package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/onesimus-systems/packet-guardian/src/common"
	"github.com/onesimus-systems/packet-guardian/src/models"
)

type User struct {
	e *common.Environment
}

func NewUserController(e *common.Environment) *User {
	return &User{e: e}
}

func (u *User) UserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		u.saveUserHandler(w, r)
	} else if r.Method == "DELETE" {
		u.deleteUserHandler(w, r)
	}
}

func (u *User) saveUserHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	if username == "" {
		common.NewAPIResponse(common.APIStatusGenericError, "Username required", nil).WriteTo(w)
		return
	}

	user, err := models.GetUserByUsername(u.e, username)
	if err != nil {
		u.e.Log.Errorf("1Error saving user: %s", err.Error())
		common.NewAPIResponse(common.APIStatusGenericError, "Error saving user", nil).WriteTo(w)
		return
	}

	// Password
	password := r.FormValue("password")
	if password != "" {
		if password == "-1" {
			user.RemovePassword()
		} else {
			user.NewPassword(password)
		}
		// } else if len(password) > 8 {
		// 	user.NewPassword(password)
		// } else {
		// 	u.e.Log.Error("2Error saving user: password too short")
		// 	common.NewAPIResponse(common.APIStatusGenericError, "Error saving user", nil).WriteTo(w)
		// 	return
		// }
	}

	// Device limit
	limitStr := r.FormValue("device_limit")
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			u.e.Log.Errorf("3Error saving user: %s", err.Error())
			common.NewAPIResponse(common.APIStatusGenericError, "Error saving user", nil).WriteTo(w)
			return
		}
		user.DeviceLimit = models.UserDeviceLimit(limit)
	}

	// Default device expiration
	expTypeStr := r.FormValue("expiration_type")
	devExpiration := r.FormValue("device_expiration")
	if expTypeStr != "" || devExpiration != "" {
		if expTypeStr == "" && devExpiration != "" {
			common.NewAPIResponse(common.APIStatusGenericError, "Error saving user: Expiration type not given", nil).WriteTo(w)
			return
		}

		expType, err := strconv.Atoi(expTypeStr)
		if err != nil {
			u.e.Log.Errorf("4Error saving user: %s", err.Error())
			common.NewAPIResponse(common.APIStatusGenericError, "Error saving user", nil).WriteTo(w)
			return
		}
		user.DeviceExpiration.Mode = models.UserExpiration(expType)
		if user.DeviceExpiration.Mode == models.UserDeviceExpirationGlobal ||
			user.DeviceExpiration.Mode == models.UserDeviceExpirationNever {
			user.DeviceExpiration.Value = 0
		} else if user.DeviceExpiration.Mode == models.UserDeviceExpirationSpecific {
			t, err := time.Parse(common.TimeFormat, devExpiration)
			if err != nil {
				u.e.Log.Errorf("5Error saving user: %s", err.Error())
				common.NewAPIResponse(common.APIStatusGenericError, "Error saving user", nil).WriteTo(w)
				return
			}
			user.DeviceExpiration.Value = t.Unix()
		} else if user.DeviceExpiration.Mode == models.UserDeviceExpirationDaily {
			secs, err := common.ParseTime(devExpiration)
			if err != nil {
				u.e.Log.Errorf("6Error saving user: %s", err.Error())
				common.NewAPIResponse(common.APIStatusGenericError, "Error saving user: Invalid time format", nil).WriteTo(w)
				return
			}
			user.DeviceExpiration.Value = secs
		} else if user.DeviceExpiration.Mode == models.UserDeviceExpirationDuration {
			d, err := time.ParseDuration(devExpiration)
			if err != nil {
				u.e.Log.Errorf("7Error saving user: %s", err.Error())
				common.NewAPIResponse(common.APIStatusGenericError, "Error saving user", nil).WriteTo(w)
				return
			}
			// time.Duration's Second() returns a float that contains the nanoseconds as well
			// For sanity we don't care and don't want the nanoseconds
			user.DeviceExpiration.Value = int64(d / time.Second)
		} else {
			u.e.Log.Error("8Error saving user: Invalid device expiration type")
			common.NewAPIResponse(common.APIStatusGenericError, "Error saving user", nil).WriteTo(w)
			return
		}
	}

	// Valid dates
	validStart := r.FormValue("valid_start")
	validEnd := r.FormValue("valid_end")
	if validStart == "0" && validEnd == "0" {
		user.ValidForever = true
		user.ValidStart = time.Unix(0, 0)
		user.ValidEnd = user.ValidStart
	} else {
		user.ValidForever = false
		if validStart != "" {
			t, err := time.Parse(common.TimeFormat, validStart)
			if err != nil {
				u.e.Log.Errorf("9Error saving user: %s", err.Error())
				common.NewAPIResponse(common.APIStatusGenericError, "Error saving user", nil).WriteTo(w)
				return
			}
			user.ValidStart = t
		}
		if validEnd != "" {
			t, err := time.Parse(common.TimeFormat, validEnd)
			if err != nil {
				u.e.Log.Errorf("10Error saving user: %s", err.Error())
				common.NewAPIResponse(common.APIStatusGenericError, "Error saving user", nil).WriteTo(w)
				return
			}
			user.ValidEnd = t
		}
		if user.ValidEnd.Before(user.ValidStart) {
			u.e.Log.Error("11Error saving user: Valid end date before start date")
			common.NewAPIResponse(common.APIStatusGenericError, "Error saving user: Valid start can't be after valid end", nil).WriteTo(w)
			return
		}
	}

	canManage := r.FormValue("can_manage")
	if canManage != "" {
		user.CanManage = (canManage == "1")
	}

	if err := user.Save(); err != nil {
		u.e.Log.Errorf("12Error saving user: %s", err.Error())
		common.NewAPIResponse(common.APIStatusGenericError, "Error saving user", nil).WriteTo(w)
		return
	}
	if user.ID == 0 {
		u.e.Log.Infof("Admin %s created user %s", models.GetUserFromContext(r).Username, user.Username)
		common.NewAPIOK("User created successfully", nil).WriteTo(w)
		return
	}
	u.e.Log.Infof("Admin %s edited user %s", models.GetUserFromContext(r).Username, user.Username)
	common.NewAPIOK("User saved successfully", nil).WriteTo(w)
}

func (u *User) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	user, err := models.GetUserByUsername(u.e, username)
	if err != nil {
		u.e.Log.Errorf("Error getting user for deletion: %s", err.Error())
		common.NewAPIResponse(common.APIStatusGenericError, "Error deleting user", nil).WriteTo(w)
		return
	}

	if err := user.Delete(); err != nil {
		u.e.Log.Errorf("Error deleting user: %s", err.Error())
		common.NewAPIResponse(common.APIStatusGenericError, "Error deleting user", nil).WriteTo(w)
		return
	}
	u.e.Log.Infof("Admin %s deleted user %s", models.GetUserFromContext(r).Username, user.Username)
	common.NewAPIOK("User deleted", nil).WriteTo(w)
}