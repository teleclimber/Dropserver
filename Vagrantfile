# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  # We base ourselves off an official Debian base box.
  config.vm.box = "debian/bookworm64"

  config.vm.network :forwarded_port, guest: 5050, host: 5050, host_ip: "127.0.0.1"

  # Create a link-local private address, so that the host can
  # use NFS with the Virtualbox guest. Virtualbox/Vagrant handles
  # network address translation so outbound network requests still
  # work.
  config.vm.provider :virtualbox do |vb, override|
    override.vm.network :private_network, ip: "192.254.254.2"
  end

  # Use a shell script to "provision" the box. This installs Dropserver from source.
  config.vm.provision "shell", inline: <<-EOF
    set -e
	echo localhost > /etc/hostname
    hostname localhost
    sudo apt-get update
    sudo DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl gpg git gcc unzip
	mkdir -p /dropsrv
	git -C "/dropsrv" pull || git clone https://github.com/teleclimber/Dropserver /dropsrv
    cd /dropsrv
	sudo mkdir -p /etc/apt/keyrings
	curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg
	NODE_MAJOR=18
	echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" | sudo tee /etc/apt/sources.list.d/nodesource.list
	sudo apt-get update
	sudo apt-get install -y nodejs
	npm install -g yarn
	curl -fsSL https://deno.land/x/install/install.sh | sh
	export DENO_INSTALL="/root/.deno"
	export PATH="$DENO_INSTALL/bin:$PATH"
	curl -L "https://go.dev/dl/go1.21.1.linux-amd64.tar.gz" -o go.tar.gz
	tar -C /usr/local -xzf go.tar.gz
	rm go.tar.gz
	echo 'export PATH=/usr/local/go/bin:$PATH' > /etc/profile.d/go.sh
	cd /dropsrv/frontend-ds-host
	yarn install && yarn run build
	cd /dropsrv
	/usr/local/go/bin/go build -ldflags="-X main.cmd_version=`git describe --tags --dirty`" -o dist/bin/ds-host ./cmd/ds-host
	mkdir -p /srv/dropserver
	mkdir -p /var/run/dropserver
	echo '{"data-dir":"/srv/dropserver","server":{"http-port": 5050,"no-tls": true},"external-access":{"scheme": "http","domain": "local.dropserver.org","subdomain": "dropid", "port": 5050},"sandbox":{"sockets-dir":"/var/run/dropserver","use-bubblewrap": false,"use-cgroups": false}}' > /etc/dropserver.json
	/dropsrv/dist/bin/ds-host -config=/etc/dropserver.json -migrate
	/dropsrv/dist/bin/ds-host -config=/etc/dropserver.json
    printf '\nYour server is online. Visit it at:'
    printf '\n  http://dropid.local.dropserver.org:5050/'
    printf '\n'
EOF

  # Calculate the number of CPUs and the amount of RAM the system has,
  # in a platform-dependent way; further logic below.
  cpus = nil
  total_kB_ram = nil

  host = RbConfig::CONFIG['host_os']
  if host =~ /darwin/
    cpus = `sysctl -n hw.ncpu`.to_i
    total_kB_ram =  `sysctl -n hw.memsize`.to_i / 1024
  elsif host =~ /linux/
    cpus = `nproc`.to_i
    total_kB_ram = `grep MemTotal /proc/meminfo | awk '{print $2}'`.to_i
  elsif host =~ /mingw/
    cpus = `powershell -Command "(Get-WmiObject Win32_Processor -Property NumberOfLogicalProcessors | Select-Object -Property NumberOfLogicalProcessors | Measure-Object NumberOfLogicalProcessors -Sum).Sum"`.to_i
    total_kB_ram = `powershell -Command "[math]::Round((Get-WmiObject -Class Win32_ComputerSystem).TotalPhysicalMemory)"`.to_i / 1024
  end

  # Use the same number of CPUs within Vagrant as the system, with 1
  # as a default.
  #
  # If we are unable to determine how much RAM the system has, use
  # 1GB. Otherwise, we aim to use 1/4 of the system RAM, with a
  # lower bound of 512MB and upper bound of 3GB. This is a compromise
  # between having the Vagrant guest operating system not run out of
  # RAM entirely (which it basically would if we went much lower than
  # 512MB) and also allowing it to use up a healthily large amount of
  # RAM so it can run faster on systems that can afford it.
  assign_cpus = nil
  assign_ram_mb = nil
  if cpus.nil? or cpus.zero?
    assign_cpus = 1
  else
    assign_cpus = cpus
  end
  if total_kB_ram.nil?
    assign_ram_mb = 1024
  else
    assign_ram_mb = (total_kB_ram / 1024 / 4)
    assign_ram_mb = [ 512, assign_ram_mb].max  # enforce lower bound
    assign_ram_mb = [3072, assign_ram_mb].min  # enforce upper bound
  end

  # Actually provide the computed CPUs/memory to the backing provider.
  config.vm.provider :virtualbox do |vb|
    vb.cpus = assign_cpus
    vb.memory = assign_ram_mb
  end
  config.vm.provider :libvirt do |libvirt|
    libvirt.cpus = assign_cpus
    libvirt.memory = assign_ram_mb
  end
end