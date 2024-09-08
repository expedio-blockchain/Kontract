kubectl create secret -n kubebuilder-system docker-registry regcred \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=expedio \
  --docker-password=dckr_pat_w3ifoB9MxAQSuhxO433GfrGtmZE \
  --docker-email=ronmegini@expedio.xyz

kubectl patch -n kubebuilder-system serviceaccount kubebuilder-controller-manager -p '{"imagePullSecrets": [{"name": "regcred"}]}'