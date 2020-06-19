Pod::Spec.new do |spec|
  spec.name         = 'Rlz'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/dsplinz2019/dsplinz'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Dsplinz Client'
  spec.source       = { :git => 'https://github.com/dsplinz2019/dsplinz.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Rlz.framework'

	spec.prepare_command = <<-CMD
    curl https://gethstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Rlz.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
