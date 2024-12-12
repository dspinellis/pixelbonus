# pixelbonus

## Installing via puppet
This repository contains a puppet script for setting up pixelbonus
on a clean installation of Debian GNU/Linux 11 (bullseye).

To use it run the following commands:

```
wget https://apt.puppet.com/puppet7-release-bullseye.deb
sudo dpkg -i puppet7-release-bullseye.deb
sudo apt update
sudo apt install puppetserver
```

Then
* Add `/opt/puppetlabs/bin/puppet` in the sudo `secure_path` option.
* Add `source /etc/profile.d/puppet-agent.sh` in ``/root/.bashrc`

Continue as follows:
```
sudo puppet module install puppetlabs-mysql
sudo puppet module install "willdurand-composer"
sudo puppet module install "puppetlabs/vcsrepo"
cd puppet
sudo puppet apply --modulepath=`sudo puppet config print modulepath`:`pwd` site.pp
```

 - Change the secret key and mailer configuration at /var/www/pixelbonus/app/config/parameters.yml
 - rm -fR /var/www/pixelbonus/app/cache/*

## Configuration
After installing pixelbonus, a few configuration parameters need to be set based on the particular setup. These include database and mailer configuration and a secret key to be used when encrypting the QR codes.

The configuration settings are located in **app/config/parameters.yml**. The file is in [YAML](https://en.wikipedia.org/wiki/YAML) format and contains the following options:
 - database_driver: One of the PHP PDO [database drivers](http://php.net/manual/en/pdo.drivers.php).
 - database_host: The database host. Usually localhost.
 - database_port: The database port. Defaults to 3306.
 - database_name: The database name. Defaults to pixelbonus.
 - database_user: The database user. Defaults to root.
 - database_password: The database password.
 - mailer_transport: One of swiftmailer's [compatible transports](http://swiftmailer.org/docs/sending.html#transport-types). Defaults to mail.
 - mailer_host: SMTP host (for smtp transport).
 - mailer_encryption: [SMTP encryption](http://swiftmailer.org/docs/sending.html#encrypted-smtp) (for smtp transport). Defaults to tls.
 - mailer_port: SMTP port (for smtp transport). Defaults to 587.
 - mailer_auth_mode: [SMTP authentication method](http://swiftmailer.org/docs/sending.html#smtp-with-a-username-and-password). Defaults to login.
 - mailer_user: SMTP user (for smtp transport).
 - mailer_password: SMTP password (for smtp transport).
 - locale: Default locale when not set explicitly by the user. Defaults to el.
 - secret: A random string used as the key for encrypting QR codes.
 - wkhtmltopdf: Absolute path to the wkhtmltopdf binary.

## Backing up
Pixelbonus stores all data in its MySQL database. Puppet can be easily set up to generate daily backups of the database. To do this run the following command in the shell of the server:

```puppet
puppet apply -e '
class { "::mysql::server":
  root_password => "",
}
class { mysql::server::backup:
  backupdatabases => [pixelbonus],
  backupdir => "/root/pixelbonus_backups",
}'
```

This will set up a cron job that runs every day at 23:05 and generates a database backup in /root/pixelbonus_backups. The resulting files can be synced to a remote host using rsync or similar utilities.

## Updating
The project is based on the Symfony framework and utilizes a number of bundles, some of which might eventually need to be updated in order to resolve bugs or security issues. To update these libraries run the following command in the shell of the server while in the project's root folder:
```bash
composer update
```
Or if composer is not in PATH, run:
```bash
php composer.phar update
```

## Try it out
To try it out without installing visit http://www.pixelbonus.com
