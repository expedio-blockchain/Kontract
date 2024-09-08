kubectl create secret -n kubebuilder-system docker-registry regcred \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=expedio \
  --docker-password=***REMOVED*** \
  --docker-email=ronmegini@expedio.xyz

kubectl patch -n kubebuilder-system serviceaccount kubebuilder-controller-manager -p '{"imagePullSecrets": [{"name": "regcred"}]}'