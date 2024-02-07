# Using grub-mkconfig

## /etc/default/grub.d/*.cfg config files


## How it works



## /etc/default/grub file

The `/etc/default/grub` is file that contains options that customizes the
`grub-mkconfig` process.
This file is actually a bash script that is
[source](https://www.gnu.org/savannah-checkouts/gnu/bash/manual/bash.html#index-source)
included by `grub-mkconfig` process.
After the script has run, `grub-mkconfig` reads environment variables that were set by
the script.

A list of the environment variables supported by `grub-mkconfig` is documented here:
[Simple configuration handling](https://www.gnu.org/software/grub/manual/grub/html_node/Simple-configuration.html#Simple-configuration)

In Mariner, the end of `/etc/default/grub` file is the following:

```bash
for x in /etc/default/grub.d/*.cfg ; do
    if [ -e "${x}" ]; then
        . "${x}"
    fi
done
```

This pulls in the `*.cfg` files in alphabetical order from the `/etc/default/grub.d/`
directory into the end of the script.
This is a semi-clever of allowing the `grub-mkconfig` environment variables to be
specified using multiple config files, which allows for some degree of modularity.




