// This source file is part of the Packet Guardian project.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
/*jslint browser:true */
/*globals $*/
$.onReady(function() {
    "use strict";
    var register = function() {
        disableRegBtn();
        var data = {
            "username": "",
            "mac-address": "",
            "description": $('[name=dev-desc]').value()
        };

        // It's not guaranteed that all fields will be shown
        // The username box will always be shown, sometimes disabled
        var username = $('[name=username]');
        if (username.length !== 0) {
            data.username = username.value();
        }
        if (data.username === "") { enableRegBtn(); return; } // Required

        // The password box will only show if the user isn't logged in
        var password = $('[name=password]');
        if (password.length !== 0) {
            data.password = password.value();
            if (data.password === "") { enableRegBtn(); return; }
        }

        // The mac-address field will only show for a manual registration
        var mac = $('[name=mac-address]');
        if (mac.length !== 0) {
            data["mac-address"] = mac.value();
            if (data["mac-address"] === "") { enableRegBtn(); return; }
        }

        // The platform field will only show for a manual registration
        var platform = $('[name=platform]');
        if (platform.length !== 0) {
            data.platform = platform.value();
            if (data.platform === "") { enableRegBtn(); return; }
        }

        if (data.password) { // Need to login first
            API.login({ "username": data.username, "password": data.password }, function(resp, req) {
                registerDevice(data);
            }, function(req) {
                window.scrollTo(0, 0);
                enableRegBtn();
                if (req.status === 401) {
                    c.FlashMessage("Incorrect username or password");
                } else {
                    c.FlashMessage("Unknown error");
                }
            });
        } else {
            registerDevice(data);
        }
    };

    function disableRegBtn() {
        $('#register-btn').prop('disabled', true);
        $('#register-btn').text("Registering...");
    }

    function enableRegBtn() {
        $('#register-btn').text("Register >");
        $('#register-btn').prop('disabled', false);
    }

    function registerDevice(data) {
        API.registerDevice(data, function(resp, req) {
            resp = JSON.parse(resp);
            window.scrollTo(0, 0);
            c.FlashMessage("Registration successful", 'success');
            $('.register-box').hide();

            if (data.password) {
                // If the use had to login to register, let's log them out.
                // It may be a bit confusing if they go back and forget they
                // had to enter a password.
                API.logout();
            }

            if (data["mac-address"] === "") {
                $('#suc-msg-auto').show();
                return;
            }

            $('#suc-msg-manual').show();
            setTimeout(function() { location.href = resp.Data.Location; }, 3000);
        }, function(req) {
            window.scrollTo(0, 0);
            enableRegBtn();
            var resp = JSON.parse(req.responseText);
            switch (req.status) {
                case 500:
                    c.FlashMessage("Internal Server Error - " + resp.Message);
                    break;
                default:
                    c.FlashMessage(resp.Message);
                    break;
            }
            if (data.password) {
                // If the user had to login to register, let's log them out.
                // It may be a bit confusing if they go back and forget they
                // had to enter a password.
                API.logout();
            }
        });
    }

    $('#register-btn').click(register);
});
