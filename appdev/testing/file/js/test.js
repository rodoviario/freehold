QUnit.module("files", {
	setup: function(assert) {
		QUnit.stop();
		//upload testing file
		var form = new FormData();
		form.append("testfile.xml", 
			new Blob(['<test id="1"><inner id="2"></inner></test>'], 
			{type: "text/xml"}), "testfile.xml");

		//upload file
		fh.file.upload("/testing/v1/file/testdata/", form)
		.always(function(result) {
			assert.deepEqual(result, {
				status: "success",
				data: [
					{
						name: "testfile.xml",
						url: "/testing/v1/file/testdata/testfile.xml"
					}
				]
			});
			QUnit.start();
		});

	},
	teardown: function(assert) {
		QUnit.stop();
		//delete file
		fh.file.delete("/testing/v1/file/testdata/testfile.xml")
		.always(function(result) {
			assert.deepEqual(result,  {
				status: "success",
				data: 
				{
					name: "testfile.xml",
					url: "/testing/v1/file/testdata/testfile.xml"
				}
			});
			QUnit.start();
		});
	}
});


QUnit.asyncTest("Set Permissions", function(assert) {
	expect(3);

	var newPrm = {
		permissions: {
			owner: fh.auth().user,
			public: "",
			private: "rw",
			friend: "r"
		}
	};

				
	
	//set permissions
	fh.properties.set("/testing/v1/file/testdata/testfile.xml", newPrm)
	.always(function(result) {
		assert.deepEqual(result, {
			"data": {
				"name": "testfile.xml",
				"url": "/testing/v1/file/testdata/testfile.xml"
			},
			"status": "success"
		});
		QUnit.start();
	});
});

QUnit.asyncTest("Get Properties", function(assert) {
	expect(3);

	fh.properties.get("/testing/v1/file/testdata/testfile.xml")
	.always(function(result) {
		assert.deepEqual(result, {
			"data": {
				"name": "testfile.xml",
				permissions: {
					owner: fh.auth().user,
					private: "rw",
				},
				"size": 42,
				"url": "/testing/v1/file/testdata/testfile.xml"
			},
			"status": "success"
		});
		QUnit.start();
	});
	
});


QUnit.module("settings");
QUnit.asyncTest("Get Setting", function(assert) {
	expect(1);

	fh.settings.get("LogErrors")
	.always(function(result) {
		//returned successfully and didn't return a map of settings
		assert.ok((result.status == "success") && 
			(result.data.description === "Whether or not errors will be logged in the core/log datastore."));
		QUnit.start();
	});
});

QUnit.asyncTest("All Settings", function(assert) {
	expect(1);

	fh.settings.all()
	.always(function(result) {
		assert.ok((result.status == "success") && 
		(result.data.LogErrors.description === "Whether or not errors will be logged in the core/log datastore."));
		QUnit.start();
	});
});


QUnit.asyncTest("Set Setting", function(assert) {
	expect(1);

	fh.settings.get("LogErrors")
	.always(function(result) {
		var val = result.data.value;
		fh.settings.set("LogErrors", false)
		.always(function(result) {
			fh.settings.get("LogErrors")
			.always(function(result) {
				assert.deepEqual(result.data.value, false);
				fh.settings.set("LogErrors", val)
				.always(function(result) {
					QUnit.start();
				});
			});
		});
	});
});

QUnit.asyncTest("Default Setting", function(assert) {
	expect(1);

	fh.settings.get("LogErrors")
	.always(function(result) {
		var val = result.data.value;
		fh.settings.default("LogErrors")
		.always(function(result) {
			fh.settings.get("LogErrors")
			.always(function(result) {
				assert.deepEqual(result.data.value, true);
				
				fh.settings.set("LogErrors", val)
				.always(function(result) {
					QUnit.start();
				});
			});
		});
	});
});


QUnit.module("Users", {
	setup: function(assert) {
		QUnit.stop();
		//create test user

	},
	teardown: function(assert) {
		QUnit.stop();
		//delete test user
	}
});


