//
// Copyright (c) 2016 Intel Corporation
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
//

package image_bat

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/01org/ciao/bat"
	"github.com/01org/ciao/ssntp/uuid"
)

const standardTimeout = time.Second * 300

func createRandomFile(sizeMB int) (path string, err error) {
	var f *os.File
	f, err = ioutil.TempFile("/tmp", "image-")
	if err != nil {
		return
	}
	defer func() {
		err1 := f.Close()
		if err1 != nil && err == nil {
			err = err1
		}
	}()

	b := make([]byte, sizeMB*1000000)
	_, err = rand.Read(b)
	if err != nil {
		return
	}
	_, err = f.Write(b)
	if err == nil {
		path = f.Name()
	}

	return
}

func addRandomImage(ctx context.Context, tenant string, size int, options *bat.ImageOptions) (*bat.Image, error) {
	path, err := createRandomFile(size)
	if err != nil {
		return nil, fmt.Errorf("Unable to create random file : %v", err)
	}
	defer func() { _ = os.Remove(path) }()
	return bat.AddImage(ctx, tenant, path, options)
}

func uploadRandomImage(ctx context.Context, tenant, id string, size int) error {
	path, err := createRandomFile(size)
	if err != nil {
		return fmt.Errorf("Unable to create random file : %v", err)
	}
	defer func() { _ = os.Remove(path) }()
	return bat.UploadImage(ctx, tenant, id, path)
}

// Add a new image, check it's listed and delete it
//
// TestAddShowDelete adds a new image containing random content to the image
// service.  It then retrieves the meta data for the new image and checks that
// various fields are correct.  Finally, it deletes the image.
//
// The image is successfully uploaded, it appears when ciao-cli image show is
// executed, it can be successfully deleted and is no longer present in the
// ciao-cli image list output after deletion.
func TestAddShowDelete(t *testing.T) {
	const name = "test-image"
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	// TODO:  The only options currently supported by the image service are
	// ID and Name.  This code needs to be updated when the image service's
	// support for meta data improves.

	options := bat.ImageOptions{
		Name: name,
	}
	img, err := addRandomImage(ctx, "", 10, &options)
	if err != nil {
		t.Fatalf("Unable to add image %v", err)
	}

	if img.ID == "" || img.Name != name || img.Status != "active" ||
		img.Visibility != "public" || img.Protected {
		t.Errorf("Meta data of added image is incorrect")
	}

	if img.ID != "" {
		gotImg, err := bat.GetImage(ctx, "", img.ID)
		if err != nil {
			t.Errorf("Unable to retrieve meta data for image %v", err)
		} else if gotImg.ID != img.ID || gotImg.Name != img.Name {
			t.Errorf("Unexpected meta data retrieved for image")
		}

		err = bat.DeleteImage(ctx, "", img.ID)
		if err != nil {
			t.Fatalf("Unable to delete image %v", err)
		}

		_, err = bat.GetImage(ctx, "", img.ID)
		if err == nil {
			t.Fatalf("Call to get non-existing image should fail")
		}
	}
}

// Delete a non-existing image
//
// TestDeleteNonExisting attempts to delete an non-existing image.
//
// The attempt to delete the non-existing image should fail
func TestDeleteNonExisting(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	err := bat.DeleteImage(ctx, "", uuid.Generate().String())
	cancelFunc()
	if err == nil {
		t.Errorf("Call to delete non-existing image should fail")
	}
}

// Check image list works correctly
//
// TestImageList retrieves the number of images in the image service, adds a new
// image, retrieves the image list once more, and then deletes the newly added image.
//
// The meta data received for each image should be correct, the meta data for the
// image should be present and the list of images returned by the image service
// should increase by 1 after the image has been added.  The image should be
// destroyed without error.
func TestImageList(t *testing.T) {
	const name = "test-image"
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()
	count, err := bat.GetImageCount(ctx, "")
	if err != nil {
		t.Fatalf("Unable to count number of images: %v", err)
	}

	options := bat.ImageOptions{
		Name: name,
	}
	img, err := addRandomImage(ctx, "", 10, &options)
	if err != nil {
		t.Fatalf("Unable to add image %v", err)
	}

	images, err := bat.GetImages(ctx, "")
	if err != nil {
		t.Errorf("Unable to retrieve image list: %v", err)
	}

	if len(images) != count+1 {
		t.Errorf("Unexpected number of images, expected %d got %d", count+1, len(images))
	}

	foundNewImage := false
	for k, newImg := range images {
		foundNewImage = k == img.ID
		if foundNewImage {
			if newImg.ID == "" || newImg.Name != name || newImg.Status != "active" ||
				newImg.Visibility != "public" || newImg.Protected {
				t.Errorf("Meta data of added image is incorrect")
			}
			break
		}
	}

	if !foundNewImage {
		t.Errorf("New image was not returned by ciao-cli image list")
	}

	err = bat.DeleteImage(ctx, "", img.ID)
	if err != nil {
		t.Fatalf("Unable to delete image %v", err)
	}
}

// Overwrite a non-existing image
//
// TestUploadNonExisting attempts to overwrite the contents of an non-existing image.
//
// The attempt to upload the new file should fail
func TestUploadNonExisting(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	err := uploadRandomImage(ctx, "", uuid.Generate().String(), 10)
	cancelFunc()
	if err == nil {
		t.Errorf("Call to upload a non-existing image should fail")
	}
}

// Overwrite an existing image
//
// TestUploadImage adds a new image and then attempts to overwrite its contents with a new
// image.  We then retrieve the meta data for the new image before deleting it.
//
// The attempts to create and overwrite the new image should both succeed.  We should be
// able to retrieve the meta data for the image and determine that the size of the
// updated image is correct.  Finally, the image should be deleted successfully.
func TestUploadImage(t *testing.T) {
	const name = "test-image"
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	options := bat.ImageOptions{
		Name: name,
	}
	img, err := addRandomImage(ctx, "", 10, &options)
	if err != nil {
		t.Fatalf("Unable to add image %v", err)
	}

	err = uploadRandomImage(ctx, "", img.ID, 20)
	if err != nil {
		t.Errorf("Failed to upload image %v", err)
	}

	_, err = bat.GetImage(ctx, "", img.ID)
	if err != nil {
		t.Errorf("Unable to retrieve meta data for image %v", err)
	}

	// TODO check size has changed.  Not currently supported by image
	// service.

	err = bat.DeleteImage(ctx, "", img.ID)
	if err != nil {
		t.Fatalf("Unable to delete image %v", err)
	}
}
