package sdk

import (
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/client"
	"github.com/pkg/errors"
)

func GetImage(c *client.Client, flags CommandOpts) (types.Image, error) {
	imageID := flags.Args[0]

	if imageID == "" {
		return types.Image{}, errors.New("Missing required image UUID parameter")
	}

	image, err := c.GetImage(imageID)
	if err != nil {
		return types.Image{}, errors.Wrap(err, "Error getting image")
	}

	return image, nil
}

func GetImageList(c *client.Client, flags CommandOpts) ([]types.Image, error) {
	images, err := c.ListImages()

	if err != nil {
		return []types.Image{}, errors.Wrap(err, "Error getting list of images")
	}

	return images, nil
}
