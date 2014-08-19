// Copyright 2014 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"net/http"
	"os"
	"path"
	"path/filepath"

	"bitbucket.org/tshannon/freehold/fail"
	"bitbucket.org/tshannon/freehold/permission"
)

type Properties struct {
	Name        string                 `json:"name,omitempty"`
	Url         string                 `json:"url,omitempty"`
	Permissions *permission.Permission `json:"permissions,omitempty"`
	Size        int64                  `json:"size,omitempty"`
	WriteLimit  float64                `json:"writeLimit,omitempty"`
}

func resPathFromProperty(propertyPath string) string {
	root, resource := splitRootAndPath(propertyPath)
	if isVersion(root) {
		//strip out properties from path
		_, resource = splitRootAndPath(resource)
		return path.Join("/", root, resource)
	}
	//must be app path
	resource = resPathFromProperty(resource)
	return path.Join("/", root, resource)
}

//TODO: Encryption
// AES?
//TODO: Individual datastore Write rate limits
//TODO: Store upload date?  And Modified date?

func propertiesGet(w http.ResponseWriter, r *http.Request) {
	auth, err := authenticate(w, r)
	if errHandled(err, w) {
		return
	}

	resource := resPathFromProperty(r.URL.Path)
	filename := urlPathToFile(resource)
	file, err := os.Open(filename)
	defer file.Close()

	if os.IsNotExist(err) {
		four04(w, r)
		return
	}

	if errHandled(err, w) {
		return
	}

	info, err := file.Stat()
	if errHandled(err, w) {
		return
	}

	if !info.IsDir() {
		prm, err := permission.Get(filename)

		if errHandled(err, w) {
			return
		}
		propPrm := permission.Properties(prm)

		if !auth.canRead(prm) {
			four04(w, r)
			return
		}

		if !auth.canRead(propPrm) {
			prm = nil
		}

		respondJsend(w, &JSend{
			Status: statusSuccess,
			Data: &Properties{
				Name:        filepath.Base(file.Name()),
				Permissions: prm,
				Url:         resource,
				Size:        info.Size(),
			},
		})
		return
	}

	files, err := file.Readdir(0)
	if errHandled(err, w) {
		return
	}

	fileList := make([]Properties, 0, len(files))

	for i := range files {
		var size int64
		var filePrm *permission.Permission
		if files[i].IsDir() {
			if auth.User == nil {
				//Public can't view the existence of directories
				continue
			}
		} else {
			size = files[i].Size()
			prm, err := permission.Get(path.Join(filename, files[i].Name()))
			if errHandled(err, w) {
				return
			}
			if !auth.canRead(prm) {
				continue
			}
			propPrm := permission.Properties(prm)
			if auth.canRead(propPrm) {
				filePrm = prm
			}
		}

		fileList = append(fileList, Properties{
			Name:        filepath.Base(files[i].Name()),
			Permissions: filePrm,
			Url:         path.Join(resource, files[i].Name()),
			Size:        size,
		})
	}

	respondJsend(w, &JSend{
		Status: statusSuccess,
		Data:   fileList,
	})

}

type PropertyInput struct {
	Permissions *PermissionsInput `json:"permissions,omitempty"`
}

type PermissionsInput struct {
	Owner   *string `json:"owner,omitempty"`
	Public  *string `json:"public,omitempty"`
	Friend  *string `json:"friend,omitempty"`
	Private *string `json:"private,omitempty"`
}

// makePermission translates a partial permissions input to a full permissions type
// by filling in the unspecfied entries from the datastore
func (pi *PermissionsInput) makePermission(curPrm *permission.Permission) *permission.Permission {
	prm := *curPrm
	if pi.Owner != nil {
		prm.Owner = *pi.Owner
	}
	if pi.Public != nil {
		prm.Public = *pi.Public
	}
	if pi.Friend != nil {
		prm.Friend = *pi.Friend
	}
	if pi.Private != nil {
		prm.Private = *pi.Private
	}

	return &prm
}

func propertiesPut(w http.ResponseWriter, r *http.Request) {
	input := &PropertyInput{}

	err := parseJson(r, input)
	if errHandled(err, w) {
		return
	}

	if input.Permissions == nil {
		errHandled(fail.New("No permissions passed in.", nil), w)
		return
	}

	auth, err := authenticate(w, r)
	if errHandled(err, w) {
		return
	}

	resource := resPathFromProperty(r.URL.Path)
	filename := urlPathToFile(resource)
	file, err := os.Open(filename)
	defer file.Close()

	if os.IsNotExist(err) {
		four04(w, r)
		return
	}

	if errHandled(err, w) {
		return
	}

	info, err := file.Stat()
	if errHandled(err, w) {
		return
	}

	if !info.IsDir() {
		prm, err := permission.Get(filename)
		if errHandled(err, w) {
			return
		}

		newprm := input.Permissions.makePermission(prm)
		propPrm := permission.Properties(prm)

		if !auth.canWrite(propPrm) {
			if !auth.canRead(propPrm) {
				four04(w, r)
				return
			}

			errHandled(fail.New("You do not have owner permissions on this resource.",
				&Properties{
					Name: filepath.Base(file.Name()),
					Url:  resource,
				}), w)

			return
		}
		err = permission.Set(filename, newprm)

		if errHandled(err, w) {
			return
		}

		respondJsend(w, &JSend{
			Status: statusSuccess,
			Data: &Properties{
				Name: filepath.Base(file.Name()),
				Url:  resource,
			},
		})
		return
	}

	files, err := file.Readdir(0)
	if errHandled(err, w) {
		return
	}

	fileList := make([]Properties, 0, len(files))
	var failures []error
	status := statusSuccess

	for i := range files {
		if !files[i].IsDir() {
			child := path.Join(filename, files[i].Name())
			cRes := path.Join(resource, files[i].Name())

			prm, err := permission.Get(child)
			if errHandled(err, w) {
				return
			}
			newprm := input.Permissions.makePermission(prm)

			propPrm := permission.Properties(prm)
			if auth.canWrite(propPrm) {
				err = permission.Set(child, newprm)
				if errHandled(err, w) {
					return
				}

			} else {
				if !auth.canRead(propPrm) {
					continue
				}

				status = statusFail
				failures = append(failures, fail.New("You do not have owner permissions on this resource.",
					&Properties{
						Name: filepath.Base(files[i].Name()),
						Url:  cRes,
					}))
			}
		}
	}

	respondJsend(w, &JSend{
		Status:   status,
		Data:     fileList,
		Failures: failures,
	})
}
