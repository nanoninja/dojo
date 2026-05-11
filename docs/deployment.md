# Deployment Guide

Ce guide explique comment déployer l'API sur un serveur distant (ex: OVH VPS) depuis zéro,
avec deux approches : **Docker Swarm** (simple, recommandé pour commencer) et **Kubernetes / K3s**
(recommandé quand tu maîtrises K8s).

---

## Sommaire

1. [Prérequis](#1-prérequis)
2. [Comprendre les images Docker et les tags](#2-comprendre-les-images-docker-et-les-tags)
3. [Choisir un registry d'images](#3-choisir-un-registry-dimages)
4. [Construire et publier une image](#4-construire-et-publier-une-image)
5. [Préparer le serveur OVH](#5-préparer-le-serveur-ovh)
6. [Déploiement avec Docker Swarm](#6-déploiement-avec-docker-swarm)
7. [Déploiement avec K3s (Kubernetes)](#7-déploiement-avec-k3s-kubernetes)
8. [Automatiser avec GitHub Actions (CI/CD)](#8-automatiser-avec-github-actions-cicd)

---

## 1. Prérequis

### Sur ton poste local

- Docker Desktop installé
- `kubectl` installé (pour K3s)
- Accès SSH au serveur OVH configuré (`~/.ssh/config`)

### Sur le serveur OVH

- Ubuntu 22.04 LTS (ou Debian 12)
- Accès root ou sudo
- Port 22 (SSH), 80, 443 ouvert dans le firewall OVH

### Vérifier que SSH fonctionne

```bash
ssh user@ton-ip-ovh "echo connexion OK"
```

---

## 2. Comprendre les images Docker et les tags

### Qu'est-ce qu'un tag ?

Une image Docker est identifiée par `nom:tag`. Le tag est comme un numéro de version.

```
ton-registry/api:latest      ← dangereux en production (vague)
ton-registry/api:v1.2.3      ← version sémantique (claire)
ton-registry/api:a3f9c12     ← SHA git court (traçable)
```

### Pourquoi ne jamais utiliser `latest` en production

`latest` est réécrit à chaque build. Si un déploiement échoue, tu ne sais plus
quelle version tourne, et tu ne peux pas revenir en arrière de façon fiable.

```bash
# Mauvaise pratique
docker build -t mon-api:latest .
# → impossible de savoir quelle version est déployée dans 3 mois

# Bonne pratique — tag avec le SHA git court
docker build -t mon-api:$(git rev-parse --short HEAD) .
# → chaque build est unique et traçable dans l'historique git
```

### La convention recommandée

```bash
VERSION=$(git rev-parse --short HEAD)   # ex: a3f9c12
IMAGE=ghcr.io/ton-username/api

docker build -t $IMAGE:$VERSION -t $IMAGE:latest .
#             ↑ tag traçable          ↑ tag pratique pour "la dernière"

docker push $IMAGE:$VERSION
docker push $IMAGE:latest
```

Tu publies toujours les deux : `latest` pour la commodité, le SHA pour le rollback.

---

## 3. Choisir un registry d'images

Un **registry** est un entrepôt en ligne qui stocke tes images Docker.
Sans registry, tu ne peux pas envoyer une image de ton poste vers le serveur.

### GitHub Container Registry (recommandé, gratuit)

Intégré à GitHub, gratuit pour les repos publics et privés.

```
ghcr.io/ton-username/api:v1.0.0
```

**Connexion :**

```bash
# Génère un Personal Access Token GitHub avec les permissions : read:packages, write:packages
echo TON_TOKEN | docker login ghcr.io -u ton-username --password-stdin
```

### Docker Hub (alternative)

Le plus connu, gratuit avec une limite sur les pulls.

```
ton-username/api:v1.0.0
```

```bash
docker login
```

---

## 4. Construire et publier une image

```bash
# Variables
IMAGE=ghcr.io/ton-username/api
VERSION=$(git rev-parse --short HEAD)

# Build avec le Dockerfile de production
docker build \
  -f deployments/docker/Dockerfile.prod \
  -t $IMAGE:$VERSION \
  -t $IMAGE:latest \
  .

# Vérifier que ça fonctionne localement
docker run --rm $IMAGE:$VERSION --version

# Publier vers le registry
docker push $IMAGE:$VERSION
docker push $IMAGE:latest
```

Après le push, ton image est accessible depuis n'importe quel serveur qui
a les droits de lecture sur le registry.

---

## 5. Préparer le serveur OVH

### 5.1 Installer Docker sur le serveur

```bash
ssh user@ton-ip-ovh

# Installer Docker
curl -fsSL https://get.docker.com | sh

# Ajouter ton user au groupe docker (pour ne pas avoir à sudo)
usermod -aG docker $USER

# Vérifier
docker --version
```

### 5.2 Créer un Docker Context sur ton poste (remplace docker-machine)

Docker Context permet d'envoyer des commandes Docker à un serveur distant via SSH,
depuis ton terminal local — sans avoir à se connecter en SSH manuellement.

```bash
# Créer le contexte (une seule fois)
docker context create ovh --docker "host=ssh://user@ton-ip-ovh"

# Voir les contextes disponibles
docker context ls

# Basculer sur le serveur distant
docker context use ovh

# Toutes les commandes docker s'exécutent maintenant sur OVH
docker ps   # ← liste les containers sur le serveur OVH

# Revenir en local
docker context use default
```

### 5.3 Copier les fichiers de configuration sur le serveur

Le serveur a besoin du fichier `.env` de production et des fichiers compose.

```bash
# Créer le dossier de l'app sur le serveur
ssh user@ton-ip-ovh "mkdir -p ~/app"

# Copier les fichiers nécessaires
scp .env.production user@ton-ip-ovh:~/app/.env
scp -r deployments/ user@ton-ip-ovh:~/app/deployments/
```

---

## 6. Déploiement avec Docker Swarm

Docker Swarm est un orchestrateur intégré à Docker. Il gère les rolling updates
(mise à jour sans downtime) et le rollback natif. C'est la solution la plus proche
de ce que faisait docker-machine + Swarm.

### 6.1 Initialiser Swarm sur le serveur (une seule fois)

```bash
ssh user@ton-ip-ovh

docker swarm init
# → affiche un token pour ajouter d'autres nœuds si besoin (inutile pour un seul serveur)
```

### 6.2 Adapter le compose pour Swarm

Swarm ne supporte pas `build:` dans les services — il utilise uniquement des
images pré-construites tirées d'un registry. Le `compose.prod.yaml` est déjà
structuré pour ça, il suffit de remplacer le champ `build` par `image`.

Crée `deployments/docker/compose.swarm.yaml` :

```yaml
services:
  api:
    image: ghcr.io/ton-username/api:${VERSION:-latest}
    deploy:
      replicas: 2
      update_config:
        parallelism: 1          # met à jour 1 replica à la fois
        delay: 10s              # attend 10s entre chaque replica
        failure_action: rollback # rollback auto si le health check échoue
      rollback_config:
        parallelism: 1
        delay: 5s
```

### 6.3 Premier déploiement

```bash
# Depuis ton poste local, en utilisant le contexte OVH
docker context use ovh

VERSION=a3f9c12   # SHA de la version à déployer

docker stack deploy \
  -c deployments/docker/compose.prod.yaml \
  -c deployments/docker/compose.swarm.yaml \
  --with-registry-auth \
  api

# Vérifier que les services tournent
docker stack services api
docker service ps api_api
```

### 6.4 Mettre à jour vers une nouvelle version

```bash
docker context use ovh

NEW_VERSION=b7d2e45

# Mettre à jour uniquement le service api
docker service update \
  --image ghcr.io/ton-username/api:$NEW_VERSION \
  --with-registry-auth \
  api_api

# Suivre la progression du rolling update
docker service ps api_api
```

Swarm remplace les replicas une par une. Pendant la mise à jour, l'ancienne version
continue de servir les requêtes — il n'y a pas de downtime.

### 6.5 Rollback

```bash
# Rollback vers la version précédente (une commande)
docker service rollback api_api

# Vérifier
docker service ps api_api
```

Swarm garde en mémoire la configuration précédente du service. Le rollback
est immédiat et sans downtime, de la même façon que la mise à jour.

### 6.6 Voir les logs

```bash
docker context use ovh
docker service logs -f api_api
```

---

## 7. Déploiement avec K3s (Kubernetes)

K3s est une distribution Kubernetes légère (~100 Mo), parfaite pour un VPS OVH.
Tu as déjà les manifests dans `deployments/k8s/` — K3s les utilise directement.

### Comprendre la séparation compose / K8s

`compose.prod.yaml` et les fichiers `deployments/k8s/` sont deux mondes **complètement séparés**.
K8s ne lit jamais un fichier compose — il ne sait pas ce que c'est.

```
compose.prod.yaml           deployments/k8s/
─────────────────           ────────────────
build: (construit           image: ghcr.io/username/api:SHA
l'image localement)         (tire l'image depuis un registry)

→ utilisé pour              → utilisé pour K8s
  Docker Compose              kubectl apply
```

Le `build:` dans `compose.prod.yaml` n'est donc pas un problème — il sert
uniquement quand tu utilises Docker Compose. Pour K8s, tu construis l'image
toi-même, tu la pousses vers un registry, et K8s va la tirer depuis là.

### La structure des overlays Kustomize

```
deployments/k8s/
  base/          ← config commune à tous les environnements
  dev/           ← overlay dev  (1 replica, image dev-latest, SSLMODE disable)
  prod/          ← overlay prod (2 replicas, image SHA réel, SSLMODE require)
```

Le `prod/kustomization.yaml` contient la section `images:` qui remplace
`your-repo/api` (le placeholder dans `base/`) par ton vrai registry + tag :

```yaml
images:
  - name: your-repo/api
    newName: ghcr.io/ton-username/api
    newTag: a3f9c12   # SHA du commit déployé
```

C'est cette ligne `newTag` que tu mets à jour à chaque déploiement.

### 7.1 Installer K3s sur le serveur (une seule fois)

```bash
ssh user@ton-ip-ovh

curl -sfL https://get.k3s.io | sh -

# Vérifier
kubectl get nodes
# NAME        STATUS   ROLES                  AGE
# ovh-vps     Ready    control-plane,master   1m
```

### 7.2 Configurer kubectl sur ton poste local

```bash
# Récupérer le fichier de configuration K8s depuis le serveur
scp user@ton-ip-ovh:/etc/rancher/k3s/k3s.yaml ~/.kube/config-ovh

# Édite le fichier : remplace "127.0.0.1" par l'IP OVH
sed -i 's/127.0.0.1/ton-ip-ovh/g' ~/.kube/config-ovh

# Utiliser ce contexte
export KUBECONFIG=~/.kube/config-ovh
kubectl get nodes   # doit afficher le nœud OVH
```

### 7.3 Créer les secrets de production

Les secrets Kubernetes stockent les valeurs sensibles (.env) hors des manifests.

```bash
# Créer le namespace
kubectl apply -f deployments/k8s/base/namespace.yaml

# Créer le secret depuis ton .env de production
kubectl create secret generic api-secrets \
  --namespace=dojo \
  --from-env-file=.env.production
```

### 7.4 Premier déploiement

```bash
# Appliquer l'overlay prod (inclut automatiquement la base)
kubectl apply -k deployments/k8s/prod

# Vérifier que les pods démarrent
kubectl get pods -n dojo -w
# NAME                   READY   STATUS    RESTARTS   AGE
# api-7d9f8b-xk2p9       1/1     Running   0          30s
# api-7d9f8b-m3n7q       1/1     Running   0          30s
```

### 7.5 Mettre à jour vers une nouvelle version

```bash
NEW_VERSION=b7d2e45

# Étape 1 — build et push de la nouvelle image
docker build -f deployments/docker/Dockerfile.prod \
  -t ghcr.io/ton-username/api:$NEW_VERSION .
docker push ghcr.io/ton-username/api:$NEW_VERSION

# Étape 2 — mettre à jour le tag dans l'overlay prod
# (kustomize le fait en une commande)
cd deployments/k8s/prod
kustomize edit set image your-repo/api=ghcr.io/ton-username/api:$NEW_VERSION
cd -

# Étape 3 — appliquer
kubectl apply -k deployments/k8s/prod

# Suivre le rolling update
kubectl rollout status deployment/api -n dojo
# Waiting for deployment "api" rollout to finish: 1 out of 2 new replicas have been updated...
# deployment "api" successfully rolled out
```

Kubernetes remplace les pods un par un, en attendant que le nouveau soit `Ready`
(health check `/readyz` doit répondre 200) avant de supprimer l'ancien.

### 7.6 Rollback

```bash
# Voir l'historique des déploiements
kubectl rollout history deployment/api -n dojo
# REVISION  CHANGE-CAUSE
# 1         version a3f9c12
# 2         version b7d2e45

# Rollback vers la révision précédente
kubectl rollout undo deployment/api -n dojo

# Rollback vers une révision spécifique
kubectl rollout undo deployment/api -n dojo --to-revision=1
```

### 7.7 Voir les logs

```bash
# Logs d'un pod spécifique
kubectl logs -f deployment/api -n dojo

# Logs de tous les pods du déploiement
kubectl logs -f -l app=api -n dojo
```

---

## 8. Automatiser avec GitHub Actions (CI/CD)

Faire tout ça manuellement fonctionne, mais rapidement tu voudras que ça s'automatise :
push sur `main` → build → push image → déploiement automatique.

Voici un pipeline minimal à placer dans `.github/workflows/deploy.yml` :

```yaml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Log in to registry
        run: echo "${{ secrets.GITHUB_TOKEN }}" | docker login ghcr.io -u ${{ github.actor }} --password-stdin

      - name: Build and push image
        run: |
          VERSION=${{ github.sha | head -c 7 }}
          IMAGE=ghcr.io/${{ github.repository_owner }}/api

          docker build -f deployments/docker/Dockerfile.prod -t $IMAGE:$VERSION -t $IMAGE:latest .
          docker push $IMAGE:$VERSION
          docker push $IMAGE:latest

      - name: Deploy (Swarm)
        env:
          DOCKER_HOST: ssh://user@ton-ip-ovh
        run: |
          docker service update \
            --image ghcr.io/${{ github.repository_owner }}/api:${{ github.sha | head -c 7 }} \
            --with-registry-auth \
            api_api
```

Les secrets (`SSH_KEY`, etc.) sont configurés dans **Settings → Secrets** du repo GitHub.

---

## Résumé — Quelle option choisir ?

| Critère | Docker Swarm | K3s |
|---|---|---|
| Complexité | Faible | Moyenne |
| Rolling update | Oui | Oui |
| Rollback | `service rollback` | `rollout undo` |
| Health checks | Manuel | Automatique (readyz/livez) |
| Familiarité | Tu connaissais Swarm | Formation K8s en cours |
| Recommandation | Maintenant | Après la formation |

**Dans les deux cas, le workflow est le même :**

```
git push → build image → tag avec SHA → push registry → déployer → vérifier
                                                                   ↓ si ça fail
                                                                rollback (1 commande)
```
