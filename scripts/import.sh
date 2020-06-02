rm -rf ./internal
rm -rf ./i
mkdir -p ./internal/p
mkdir -p ./i
cp -R ../../go-platform-layer/ internal/p
cp -R ../../go-service-layer/ i
cd internal
ack -l 'go-platform-layer' | xargs perl -pi -E 's/go-platform-layer/h2\/go\/internal\/p/g'
ack -l 'go-service-layer' | xargs perl -pi -E 's/go-service-layer/h2\/go\/i/g'
cd p
rm -rf .git .gitmodules
cd ../../
cd i
ack -l 'go-platform-layer' | xargs perl -pi -E 's/go-platform-layer/h2\/go\/internal\/p/g'
ack -l 'go-service-layer' | xargs perl -pi -E 's/go-service-layer/h2\/go\/i/g'
rm -rf .git .gitmodules

cd ..

# correct service proto imports
ack -l 'go-login-service' | xargs perl -pi -E 's/go-login-service\/proto/h2\/proto\/login\/proto/g'
ack -l 'discovery-service' | xargs perl -pi -E 's/discovery-service\/proto/h2\/proto\/discovery\/proto/g'

# hailo lib cruft

mkdir -p ./internal/l
cp -R ../../go-hailo-lib/validate ./internal/l
cp -R ../../go-hailo-lib/proc ./internal/l
cp -R ../../go-hailo-lib/multierror ./internal/l
cp -R ../../go-hailo-lib/schema ./internal/l

ack -l 'go-hailo-lib' | grep import -v | xargs perl -pi -E 's/go-hailo-lib\/validate/h2\/go\/internal\/l\/validate/g'
ack -l 'go-hailo-lib' | grep import -v | xargs perl -pi -E 's/go-hailo-lib\/proc/h2\/go\/internal\/l\/proc/g'
ack -l 'go-hailo-lib' | grep import -v | xargs perl -pi -E 's/go-hailo-lib\/multierror/h2\/go\/internal\/l\/multierror/g'
ack -l 'go-hailo-lib' | grep import -v | xargs perl -pi -E 's/go-hailo-lib\/schema/h2\/go\/internal\/l\/schema/g'
