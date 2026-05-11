# Déploiement Kubernetes (K3s)

Guide complet pour déployer l'API sur un VPS OVH avec K3s, de zéro jusqu'en production.
En suivant ce guide dans l'ordre, tu seras capable de déployer, mettre à jour et rollback
sans dépendre d'une autre documentation.

---

## Sommaire

1. [Concepts clés à comprendre](#1-concepts-clés-à-comprendre)
2. [Structure des manifests du projet](#2-structure-des-manifests-du-projet)
3. [Prérequis sur ton poste local](#3-prérequis-sur-ton-poste-local)
4. [Préparer le serveur OVH](#4-préparer-le-serveur-ovh)
5. [Installer K3s](#5-installer-k3s)
6. [Connecter kubectl depuis ton poste](#6-connecter-kubectl-depuis-ton-poste)
7. [Préparer le registry d'images](#7-préparer-le-registry-dimages)
8. [Construire et publier l'image](#8-construire-et-publier-limage)
9. [Configurer les secrets](#9-configurer-les-secrets)
10. [Premier déploiement](#10-premier-déploiement)
11. [Vérifier que tout fonctionne](#11-vérifier-que-tout-fonctionne)
12. [Exposer l'API sur internet (Ingress + HTTPS)](#12-exposer-lapi-sur-internet-ingress--https)
13. [Mettre à jour vers une nouvelle version](#13-mettre-à-jour-vers-une-nouvelle-version)
14. [Rollback](#14-rollback)
15. [Commandes du quotidien](#15-commandes-du-quotidien)
16. [Automatiser avec GitHub Actions (CI/CD)](#16-automatiser-avec-github-actions-cicd)

---

## 1. Concepts clés à comprendre

Avant les commandes, voici les objets K8s utilisés dans ce projet et ce qu'ils font.

### Pod
L'unité de base. Un Pod contient un ou plusieurs conteneurs. Tu ne manipules presque
jamais les Pods directement — c'est le Deployment qui les gère pour toi.

```
Deployment → gère → ReplicaSet → gère → Pods (conteneurs)
```

### Deployment
Décrit l'état désiré de l'application. Tu dis "je veux 2 replicas de l'image X", K8s
s'assure que c'est toujours le cas. Si un Pod plante, il en recrée un automatiquement.

### Service
Adresse réseau stable qui pointe vers les Pods. Les Pods ont des IPs qui changent à
chaque redémarrage — le Service fournit une IP fixe et fait le load balancing.

```
Client → Service (IP fixe) → Pod 1
                           → Pod 2
```

### ConfigMap
Stocke les variables d'environnement non-sensibles (APP_ENV, DB_HOST...).
Modifiable sans reconstruire l'image.

### Secret
Stocke les valeurs sensibles encodées en base64 (mots de passe, clés JWT...).
N'apparaît jamais en clair dans les logs K8s.

### Namespace
Espace d'isolation logique. Tout le projet vit dans le namespace `dojo`.
Permet de ne pas mélanger les ressources avec d'autres apps sur le même cluster.

### Kustomize
Outil (intégré à kubectl) qui permet d'avoir une config de base et des overlays par
environnement, sans dupliquer les fichiers.

```
base/     ← commun à tous les environnements
dev/      ← overlay dev  : 1 replica, image dev, SSL off
prod/     ← overlay prod : 2 replicas, image réelle, SSL on
```

---

## 2. Structure des manifests du projet

```
deployments/k8s/
├── base/
│   ├── kustomization.yaml          ← liste tous les fichiers de base
│   ├── namespace.yaml              ← namespace "dojo"
│   ├── secret.yaml                 ← template de Secret (à remplir)
│   ├── app/
│   │   ├── deployment.yaml         ← Deployment de l'API (2 replicas, health checks)
│   │   └── service.yaml            ← Service ClusterIP (80 → 8000)
│   ├── postgres/
│   │   ├── statefulset.yaml        ← PostgreSQL (StatefulSet pour l'identité stable)
│   │   ├── service.yaml            ← Service interne postgres:5432
│   │   └── pvc.yaml                ← PersistentVolumeClaim (disque pour les données)
│   └── redis/
│       ├── deployment.yaml         ← Redis
│       └── service.yaml            ← Service interne redis:6379
├── dev/
│   ├── kustomization.yaml          ← patch : 1 replica, image dev-latest
│   └── configmap.yaml              ← variables dev (DB_SSLMODE: disable, etc.)
└── prod/
    ├── kustomization.yaml          ← patch : image ghcr.io/username/api:SHA
    └── configmap.yaml              ← variables prod (DB_SSLMODE: require, etc.)
```

---

## 3. Prérequis sur ton poste local

```bash
# Docker Desktop
docker --version

# kubectl — outil CLI pour piloter K8s
# macOS
brew install kubectl

# Vérifier
kubectl version --client
```

---

## 4. Préparer le serveur OVH

### Accès SSH

Assure-toi que la connexion SSH fonctionne sans mot de passe (clé SSH) :

```bash
ssh-copy-id user@ton-ip-ovh
ssh user@ton-ip-ovh "echo connexion OK"
```

### Firewall OVH

Dans l'interface OVH, ouvre les ports suivants :

| Port | Protocole | Usage |
|------|-----------|-------|
| 22   | TCP | SSH |
| 80   | TCP | HTTP (redirigé vers HTTPS) |
| 443  | TCP | HTTPS |
| 6443 | TCP | API K8s (kubectl depuis ton poste) |

---

## 5. Installer K3s

K3s est un Kubernetes complet en un seul binaire. L'installation prend moins d'une minute.

```bash
# Se connecter au serveur
ssh user@ton-ip-ovh

# Installer K3s
curl -sfL https://get.k3s.io | sh -

# Vérifier que le nœud est Ready
kubectl get nodes
# NAME        STATUS   ROLES                  AGE   VERSION
# ovh-vps     Ready    control-plane,master   1m    v1.31.x
```

K3s installe et démarre automatiquement :
- `k3s-server` (le plan de contrôle K8s)
- `containerd` (le runtime de conteneurs, remplace Docker dans K8s)
- `traefik` (Ingress controller — sert à exposer les services sur internet)

---

## 6. Connecter kubectl depuis ton poste

Par défaut, `kubectl` pointe sur ton cluster local. On va lui dire de pointer
sur le serveur OVH.

```bash
# Récupérer le fichier de config K8s depuis le serveur
scp user@ton-ip-ovh:/etc/rancher/k3s/k3s.yaml ~/.kube/config-ovh

# Remplacer 127.0.0.1 par l'IP publique OVH dans ce fichier
sed -i '' 's/127.0.0.1/ton-ip-ovh/g' ~/.kube/config-ovh
# Note : sur Linux, sed -i (sans '')

# Dire à kubectl d'utiliser ce fichier
export KUBECONFIG=~/.kube/config-ovh
# Pour que ce soit permanent, ajoute cette ligne dans ton ~/.zshrc ou ~/.bashrc

# Vérifier
kubectl get nodes
# NAME        STATUS   ROLES                  AGE
# ovh-vps     Ready    control-plane,master   5m
```

---

## 7. Préparer le registry d'images

Un **registry** est un entrepôt en ligne pour tes images Docker. Le serveur K8s
va tirer l'image depuis ce registry lors du déploiement.

On utilise **GitHub Container Registry (ghcr.io)** — gratuit, intégré à GitHub.

### Créer un Personal Access Token GitHub

1. GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Génère un token avec les permissions : `read:packages`, `write:packages`, `delete:packages`
3. Sauvegarde le token (il n'est affiché qu'une fois)

### Se connecter au registry depuis ton poste

```bash
echo TON_TOKEN | docker login ghcr.io -u ton-username --password-stdin
# Login Succeeded
```

### Autoriser K3s à tirer les images privées

K3s a besoin des identifiants pour puller l'image depuis ghcr.io.

```bash
# Créer le fichier de credentials sur le serveur
ssh user@ton-ip-ovh "sudo mkdir -p /etc/rancher/k3s"

cat <<EOF | ssh user@ton-ip-ovh "sudo tee /etc/rancher/k3s/registries.yaml"
mirrors:
  ghcr.io:
    endpoint:
      - "https://ghcr.io"
configs:
  "ghcr.io":
    auth:
      username: ton-username
      password: TON_TOKEN
EOF

# Redémarrer K3s pour prendre en compte
ssh user@ton-ip-ovh "sudo systemctl restart k3s"
```

---

## 8. Construire et publier l'image

Avant chaque déploiement, tu construis l'image et tu la pousses vers ghcr.io.

```bash
# Variables — à adapter
REGISTRY=ghcr.io/ton-username
IMAGE=$REGISTRY/api
VERSION=$(git rev-parse --short HEAD)   # ex: a3f9c12

# Build avec le Dockerfile de production
docker build \
  -f deployments/docker/Dockerfile.prod \
  -t $IMAGE:$VERSION \
  -t $IMAGE:latest \
  .

# Pousser vers ghcr.io
docker push $IMAGE:$VERSION
docker push $IMAGE:latest

echo "Image publiée : $IMAGE:$VERSION"
```

**Pourquoi tagger avec le SHA git ?**
Chaque commit a un SHA unique. En utilisant ce SHA comme tag d'image, tu peux
toujours savoir exactement quel code tourne en production, et revenir à n'importe
quelle version précédente.

---

## 9. Configurer les secrets

Les secrets K8s stockent les valeurs sensibles. Ils ne doivent **jamais** être
commités dans git avec de vraies valeurs.

### Encoder une valeur en base64

```bash
echo -n "ma-valeur-secrete" | base64
# bWEtdmFsZXVyLXNlY3JldGU=
```

### Créer le namespace (une seule fois)

```bash
kubectl apply -f deployments/k8s/base/namespace.yaml
```

### Créer le Secret de production

Ne modifie pas `base/secret.yaml` directement. Crée le Secret directement
en ligne de commande (il ne sera jamais stocké dans git) :

```bash
kubectl create secret generic api-secret \
  --namespace=dojo \
  --from-literal=APP_ENCRYPTION_KEY="une-clé-de-exactement-32-caractères!!" \
  --from-literal=JWT_SECRET="un-secret-jwt-long-et-aléatoire-minimum-32-chars" \
  --from-literal=DB_PASSWORD="mot-de-passe-postgres" \
  --from-literal=REDIS_PASSWORD="" \
  --from-literal=SMTP_PASSWORD="mot-de-passe-smtp"

# Vérifier (les valeurs n'apparaissent pas en clair)
kubectl get secret api-secret -n dojo
```

> **Astuce :** pour générer une clé de 32 caractères aléatoire :
> ```bash
> openssl rand -hex 16   # → 32 caractères hex
> ```

---

## 10. Premier déploiement

### Mettre à jour le tag d'image dans l'overlay prod

```bash
VERSION=$(git rev-parse --short HEAD)

# kustomize met à jour newTag dans prod/kustomization.yaml automatiquement
cd deployments/k8s/prod
kustomize edit set image your-repo/api=ghcr.io/ton-username/api:$VERSION
cd -
```

Ce que fait cette commande : elle modifie `prod/kustomization.yaml` pour mettre
`newTag: a3f9c12` (ou le SHA de ton commit). C'est le seul fichier qui change
entre deux déploiements.

### Appliquer les manifests

```bash
kubectl apply -k deployments/k8s/prod
```

Une seule commande applique tout : namespace, secrets template, ConfigMap, Deployment,
Services, StatefulSet PostgreSQL, Redis — dans le bon ordre.

### Vérifier le démarrage

```bash
# Voir l'état de tous les pods en temps réel (-w = watch)
kubectl get pods -n dojo -w

# Ce que tu dois voir au bout de ~30 secondes
# NAME                    READY   STATUS    RESTARTS   AGE
# api-7d9f8b-xk2p9        1/1     Running   0          45s
# api-7d9f8b-m3n7q        1/1     Running   0          45s
# postgres-0              1/1     Running   0          45s
# redis-6c8d9f-p4n2m      1/1     Running   0          45s
```

Si un pod est en `CrashLoopBackOff` ou `Error`, voir la section [diagnostiquer un problème](#diagnostiquer-un-problème).

---

## 11. Vérifier que tout fonctionne

```bash
# Tester l'API depuis le cluster (sans Ingress, l'API n'est pas encore exposée sur internet)
kubectl port-forward service/api 8080:80 -n dojo
# → l'API est accessible sur http://localhost:8080 depuis ton poste

curl http://localhost:8080/health
# {"status":"ok","version":"..."}

# Couper le port-forward avec Ctrl+C
```

---

## 12. Exposer l'API sur internet (Ingress + HTTPS)

### Comprendre l'Ingress

Le Service `api` est de type `ClusterIP` — accessible uniquement à l'intérieur du
cluster. Pour l'exposer sur internet, on utilise un **Ingress** : une règle de
routage HTTP qui dit "les requêtes vers `api.ton-domaine.com` vont au Service `api`".

K3s installe **Traefik** comme Ingress controller par défaut — il écoute sur les
ports 80 et 443 du serveur.

### Créer le fichier Ingress

Crée `deployments/k8s/prod/ingress.yaml` :

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api
  namespace: dojo
  annotations:
    # cert-manager crée automatiquement le certificat TLS
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  rules:
    - host: api.ton-domaine.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api
                port:
                  number: 80
  tls:
    - hosts:
        - api.ton-domaine.com
      secretName: api-tls   # K8s stocke le certificat ici
```

Ajoute ce fichier dans `prod/kustomization.yaml` :

```yaml
resources:
  - ../base
  - configmap.yaml
  - ingress.yaml    # ← ajouter
```

### Installer cert-manager (HTTPS automatique Let's Encrypt)

```bash
# Installer cert-manager dans le cluster
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml

# Attendre que les pods cert-manager soient Running
kubectl get pods -n cert-manager -w
```

Crée `deployments/k8s/prod/clusterissuer.yaml` :

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: ton-email@example.com   # ← pour les alertes d'expiration
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - http01:
          ingress:
            class: traefik
```

```bash
kubectl apply -f deployments/k8s/prod/clusterissuer.yaml
kubectl apply -k deployments/k8s/prod
```

Après ~60 secondes, `https://api.ton-domaine.com` est accessible avec un
certificat Let's Encrypt valide, renouvelé automatiquement.

> **Prérequis DNS** : le domaine `api.ton-domaine.com` doit pointer vers l'IP
> du serveur OVH avant d'appliquer l'Ingress.

---

## 13. Mettre à jour vers une nouvelle version

Workflow complet à chaque nouvelle version :

```bash
# 1. S'assurer d'être sur le bon commit
git log --oneline -5

# 2. Définir la version
VERSION=$(git rev-parse --short HEAD)
IMAGE=ghcr.io/ton-username/api

# 3. Construire et pousser l'image
docker build -f deployments/docker/Dockerfile.prod -t $IMAGE:$VERSION -t $IMAGE:latest .
docker push $IMAGE:$VERSION
docker push $IMAGE:latest

# 4. Mettre à jour le tag dans l'overlay prod
cd deployments/k8s/prod
kustomize edit set image your-repo/api=$IMAGE:$VERSION
cd -

# 5. Commiter le changement de tag (traçabilité)
git add deployments/k8s/prod/kustomization.yaml
git commit -m "deploy: $VERSION"

# 6. Appliquer
kubectl apply -k deployments/k8s/prod

# 7. Suivre le rolling update
kubectl rollout status deployment/api -n dojo
# Waiting for deployment "api" rollout to finish: 1 out of 2 new replicas have been updated...
# Waiting for deployment "api" rollout to finish: 1 old replicas are pending termination...
# deployment "api" successfully rolled out
```

**Comment fonctionne le rolling update :**
K8s démarre un nouveau Pod avec la nouvelle image, attend que `/readyz` réponde 200,
puis arrête un ancien Pod. Il répète jusqu'à ce que tous les Pods soient mis à jour.
Il n'y a aucune interruption de service.

---

## 14. Rollback

### Rollback vers la version précédente

```bash
kubectl rollout undo deployment/api -n dojo

# Vérifier que le rollback est terminé
kubectl rollout status deployment/api -n dojo

# Voir quelle image tourne maintenant
kubectl get deployment api -n dojo -o jsonpath='{.spec.template.spec.containers[0].image}'
```

### Voir l'historique des déploiements

```bash
kubectl rollout history deployment/api -n dojo
# REVISION  CHANGE-CAUSE
# 1         <none>
# 2         <none>
# 3         <none>
```

### Rollback vers une révision spécifique

```bash
# Voir le détail d'une révision
kubectl rollout history deployment/api -n dojo --revision=2

# Revenir à cette révision précise
kubectl rollout undo deployment/api -n dojo --to-revision=2
```

> **Note :** si tu commites le changement de `kustomization.yaml` à chaque
> déploiement (étape 5 du workflow), tu peux aussi faire un rollback via git :
> `git revert` sur le commit de tag, puis `kubectl apply -k`.

---

## 15. Commandes du quotidien

### Voir l'état général

```bash
# Tous les pods du namespace
kubectl get pods -n dojo

# Tous les services
kubectl get services -n dojo

# Toutes les ressources d'un coup
kubectl get all -n dojo
```

### Logs

```bash
# Logs de l'API (suit en temps réel)
kubectl logs -f deployment/api -n dojo

# Logs de tous les pods api en même temps
kubectl logs -f -l app=api -n dojo

# Logs des 100 dernières lignes
kubectl logs deployment/api -n dojo --tail=100

# Logs de PostgreSQL
kubectl logs statefulset/postgres -n dojo
```

### Diagnostiquer un problème

```bash
# Un pod ne démarre pas — voir les events
kubectl describe pod <nom-du-pod> -n dojo
# Cherche la section "Events" en bas — c'est là que les erreurs apparaissent

# Exemples d'erreurs courantes :
# "ImagePullBackOff"  → K3s ne peut pas tirer l'image (vérifier les credentials registry)
# "CrashLoopBackOff"  → le conteneur plante au démarrage (vérifier les logs)
# "Pending"           → pas de ressources disponibles sur le nœud
```

### Entrer dans un pod (debug)

```bash
kubectl exec -it deployment/api -n dojo -- sh
```

### Redémarrer un déploiement sans changer l'image

```bash
kubectl rollout restart deployment/api -n dojo
```

### Voir la consommation de ressources

```bash
kubectl top pods -n dojo
# NAME                    CPU(cores)   MEMORY(bytes)
# api-7d9f8b-xk2p9        5m           32Mi
# postgres-0              8m           64Mi
```

---

## 16. Automatiser avec GitHub Actions (CI/CD)

Plutôt que de faire les étapes manuellement, on automatise : chaque push sur `main`
déclenche le build, le push et le déploiement.

Crée `.github/workflows/deploy.yml` dans ton repo :

```yaml
name: Deploy to production

on:
  push:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE: ghcr.io/${{ github.repository_owner }}/api

jobs:
  deploy:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Get short SHA
        id: sha
        run: echo "short=$(git rev-parse --short HEAD)" >> $GITHUB_OUTPUT

      - name: Build and push image
        uses: docker/build-push-action@v5
        with:
          context: .
          file: deployments/docker/Dockerfile.prod
          push: true
          tags: |
            ${{ env.IMAGE }}:${{ steps.sha.outputs.short }}
            ${{ env.IMAGE }}:latest

      - name: Update image tag in kustomization
        run: |
          cd deployments/k8s/prod
          kustomize edit set image your-repo/api=${{ env.IMAGE }}:${{ steps.sha.outputs.short }}
          cd -
          git config user.name "github-actions"
          git config user.email "actions@github.com"
          git add deployments/k8s/prod/kustomization.yaml
          git commit -m "deploy: ${{ steps.sha.outputs.short }}" || echo "nothing to commit"
          git push

      - name: Deploy to K3s
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.OVH_HOST }}
          username: ${{ secrets.OVH_USER }}
          key: ${{ secrets.OVH_SSH_KEY }}
          script: |
            cd ~/app
            git pull
            kubectl apply -k deployments/k8s/prod
            kubectl rollout status deployment/api -n dojo --timeout=120s
```

### Secrets GitHub à configurer

Dans **Settings → Secrets and variables → Actions** de ton repo :

| Secret | Valeur |
|--------|--------|
| `OVH_HOST` | IP de ton serveur OVH |
| `OVH_USER` | Utilisateur SSH (ex: `ubuntu`) |
| `OVH_SSH_KEY` | Contenu de ta clé privée SSH (`~/.ssh/id_rsa`) |

`GITHUB_TOKEN` est automatiquement fourni par GitHub, pas besoin de le créer.

### Ce qui se passe à chaque push sur main

```
git push
  → GitHub Actions démarre
  → build image Docker
  → push vers ghcr.io:SHA
  → met à jour prod/kustomization.yaml
  → se connecte en SSH au serveur OVH
  → git pull (récupère le nouveau kustomization.yaml)
  → kubectl apply -k deployments/k8s/prod
  → rolling update automatique
  → attend que le deployment soit stable (rollout status)
  → si ça échoue → le job GitHub Actions passe en rouge
```

En cas d'échec du déploiement, le rollback se fait en une commande depuis ton poste :

```bash
export KUBECONFIG=~/.kube/config-ovh
kubectl rollout undo deployment/api -n dojo
```
