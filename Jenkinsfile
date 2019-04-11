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
            }
        }
        stage('Push [local registry]') {
            steps {
                sh 'make push-development'
            }
        }
        stage('Tests') {
            steps {
                sh 'No tests found'
                sh 'exit 1'
            }
        }
        stage('Build production') {
            steps {
                sh 'make build-production'
            }
        }
        stage('Push [hub.docker.com]') {
            when {
                branch 'master'
            }
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
