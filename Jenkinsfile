pipeline {
    options {
        disableConcurrentBuilds()
    }
    agent {
        node {
            label 'solutions-126'
        }
    }
    stages {
        stage('Build development') {
            steps {
                sh 'make build-development'
                sh 'make uninstall-development'
            }
        }
        stage('Push [local registry]') {
            steps {
                sh 'make push-development'
            }
        }
        stage('Build production') {
            steps {
                sh 'make build-production'
                sh 'make uninstall-production'
            }
        }
        stage('Push [hub.docker.com]') {
            environment {
                DOCKER = credentials('docker-hub-credentials')
            }
            steps {
                sh '''
                    docker login -u ${DOCKER_USR} -p ${DOCKER_PSW};
                    make push-production;
                '''
            }
        }
    }
}
