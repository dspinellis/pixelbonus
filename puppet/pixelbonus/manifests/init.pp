class pixelbonus (
    $user = 'www-data',
    $group = 'www-data',
    $repo_url = 'https://github.com/dnna/pixelbonus.git',
    $apt_update_threshold = 2419200
) {

    # execute 'apt-get update'
      exec { 'apt-update':                    # exec resource named 'apt-update'
      command => '/usr/bin/apt-get update',  # command this resource will run
      onlyif => "/bin/bash -c 'exit $(( $(( $(date +%s) - $(stat -c %Y /var/lib/apt/lists/$( ls /var/lib/apt/lists/ -tr1|tail -1 )) )) <= ${apt_update_threshold} ))'" # Only update if repo older than a month
    }

    # install apache2 package
    package { 'apache2':
      require => Exec['apt-update'],        # require 'apt-update' before installing
      ensure => installed,
    }

    exec { 'allow-override':
      command => "sed -i 's/AllowOverride None/AllowOverride All/g' /etc/apache2/apache2.conf",
      path    => '/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin',
      cwd     => '/etc/apache2',
      require => [ Package['apache2'] ],
      onlyif  => 'grep -c "AllowOverride None" /etc/apache2/apache2.conf',
    }

    exec { 'enable-mod-rewrite':
      command => "a2enmod rewrite; /etc/init.d/apache2 restart",
      path    => '/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin',
      cwd     => '/etc/apache2',
      require => [ Exec['allow-override'] ],
      unless  => 'apachectl -t -D DUMP_MODULES |grep -c rewrite',
    }

    # ensure apache2 service is running
    service { 'apache2':
      ensure => running,
      require => Exec['enable-mod-rewrite']
    }

    # install mariadb-server package
    package { 'mariadb-server':
      require => Exec['apt-update'],        # require 'apt-update' before installing
      ensure => installed,
    }

    # ensure mysql service is running
    service { 'mysql':
      ensure => running,
      require => [ Package['mariadb-server'] ],
    }

    # install php package
    package { 'php':
      require => Exec['apt-update'],        # require 'apt-update' before installing
      ensure => installed,
    }
    package { 'php-mysql':
      require => Exec['apt-update'],        # require 'apt-update' before installing
      ensure => installed,
    }

    mysql::db { 'pixelbonus':
      user     => 'pixelbonus',
      password => 'pixelbonus',
      host     => 'localhost',
      grant    => ['SELECT', 'UPDATE'],
      require => [ Service['mysql'] ],
    }

    # install git package
    package { 'git':
      require => Exec['apt-update'],        # require 'apt-update' before installing
      ensure => installed,
    }

    file { 'ensure-vcs-folder-permissions':
      path   => '/var/www/pixelbonus',
      ensure => 'directory',
      recurse => true,
      owner => $user,
      group => $group,
    }

    file { 'ensure-cache-folder-permissions':
      path   => '/usr/local/bin/cache',
      ensure => 'directory',
      recurse => true,
      owner => $user,
      group => $group,
    }

    vcsrepo { "/var/www/pixelbonus":
        ensure   => latest,
        user     => $user,
        owner    => $user,
        group    => $group,
        provider => git,
        require  => [ Package["git"], File['ensure-vcs-folder-permissions'], File['ensure-cache-folder-permissions'] ],
        source   => $repo_url,
        revision => 'master',
    }

    package { 'wkhtmltopdf':
      require => Exec['apt-update'],        # require 'apt-update' before installing
      ensure => installed,
    }
    package { 'xvfb':
      require => Exec['apt-update'],        # require 'apt-update' before installing
      ensure => installed,
    }

    exec { 'composer-install':
      environment => [ 'COMPOSER_HOME=/usr/local/lib/composer' ],
      command     => '/usr/bin/curl -sS https://getcomposer.org/installer | /usr/bin/php && mv composer.phar /usr/local/bin/composer',
      require     => [Package['php']],
      logoutput   => on_failure,
      creates     => '/usr/local/bin/composer',
    }

    exec { 'composer-update':
      command => "composer update --no-interaction",
      path    => '/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin',
      cwd     => '/var/www/pixelbonus',
      environment => [ 'COMPOSER_HOME=/usr/local/lib/composer' ],
      user    => $user,
      refreshonly => true,
      subscribe => Vcsrepo['/var/www/pixelbonus'],
      require => [
        Vcsrepo['/var/www/pixelbonus'],
        Package['wkhtmltopdf'],
        Package['xvfb'],
        Exec['composer-install'],
        ],
      tries => 10,
      try_sleep => 5,
    }

    file { 'document-root':
      path   => '/var/www/html',
      force  => true,
      ensure => 'link',
      owner => $user,
      group => $group,
      target => '/var/www/pixelbonus/web',
      require => [ Vcsrepo['/var/www/pixelbonus'] ],
    }

    exec { 'schema-update':
      command => "php app/console doctrine:schema:update --force",
      path    => '/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin',
      cwd     => '/var/www/pixelbonus',
      user    => $user,
      refreshonly => true,
      subscribe => Exec['composer-update'],
      require => [ Exec['composer-update'] ],
    }

    cron { generateqr:
      command => "/usr/bin/php /var/www/pixelbonus/app/console pixelbonus:generateqr --env=prod --no-debug",
      user    => $user,
      minute  => "*/1",
      require => [ Exec['schema-update'] ],
    }
}
