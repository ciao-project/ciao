# Contributing to Ciao

Ciao is an open source project licensed under the [Apache v2 License] (https://opensource.org/licenses/Apache-2.0)

## Coding Style

Ciao uses the golang coding style, go fmt is your friend.

## Certificate of Origin

In order to get a clear contribution chain of trust we use the [signed-off-by language] (https://01.org/community/signed-process)
used by the Linux kernel project.

## Patch format

Beside the signed-off-by footer, we expect each patch to comply with the following format:

```
       <component>: Change summary

       More detailled explanation of your changes: Why and how.
       Wrap it to 72 characters.
       See [here] (http://chris.beams.io/posts/git-commit/)
       for some more good advices.

       Signed-off-by: <contributor@foo.com>
```

For example:

```
	ssntp: Implement role checking

	SSNTP roles are contained within the SSNTP certificates
	as key extended attributes. On both the server and client
	sides we are verifying that the claimed roles through the
	SSNTP connection protocol match the certificates.

	Signed-off-by: Samuel Ortiz <sameo@linux.intel.com>
```

## Pull requests

We accept github pull requests.