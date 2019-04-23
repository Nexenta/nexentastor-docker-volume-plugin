pipeline {
    parameters {
        string(name: 'TEST_DOCKER_IP', defaultValue: '10.3.199.249', description: 'Docker setup IP address to test on', trim: true)
    }
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
        stage('Tests [unit]') {
            steps {
                sh 'make test-unit-container'
            }
        }
        stage('Tests [e2e-docker]') {
            steps {
                sh 'TEST_DOCKER_IP=${TEST_DOCKER_IP} make test-e2e-docker-development'
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
