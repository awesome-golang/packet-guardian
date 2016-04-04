j.OnReady(function() {
    j.Click('[name=add-device-btn]', function() {
        // Redirect to manual registration page
    });
    j.Click('[name=del-selected-btn]', function() {
        var checked = j.$('.device-select:checked', true);
        var devicesToRemove = [];
        for (var i = 0; i < checked.length; i++) {
            devicesToRemove.push(checked[i].value);
        }
        j.Post('/devices/delete', {"devices": devicesToRemove}, function(resp) {
            resp = JSON.parse(resp);
            if (resp.Code === 0) {
                location.reload();
                return;
            }
            c.FlashMessage("Error deleteing devices");
        });
    });
    j.Click('[name=del-all-btn]', function() {
        j.Post('/devices/delete', {}, function(resp) {
            resp = JSON.parse(resp);
            if (resp.Code === 0) {
                location.reload();
                return;
            }
            c.FlashMessage("Error deleteing devices");
        });
    });
});
