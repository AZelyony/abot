pipeline {
    agent any
    parameters {
        choice(name: 'OS', choices: ['linux', 'apple', 'windows'], description: 'Choice OS')
        choice(name: 'ARCH', choices: ['amd64', 'arm64'], description: 'Choice ARCH')
    }


    environment {
        REPO = 'https://github.com/AZelyony/abot'
        BRANCH = 'develop'
    }

    stages {

        stage('clone') {
            steps {
                echo 'Clone Repository'
                git branch: "${BRANCH}", url: "${REPO}"
            }
        }

        stage('test') {
            steps {
                echo 'Testing started'
                sh "make test"
            }
        }

        stage('build') {
            steps {
                echo "Building binary for platform ${params.OS} on ${params.ARCH} started"
                sh "make build TARGETOS=${params.OS} TARGETARCH=${params.ARCH}"
            }
        }

        stage('image') {
            steps {
                echo "Building image for platform ${params.OS} on ${params.ARCH} started"
                sh "make image TARGETOS=${params.OS} TARGETARCH=${params.ARCH}"
            }
        }

        stage('push image') {
            steps {
			    script {
				    docker.withRegistry( '', 'dockerhub' ){
				    sh "make push TARGETOS=${params.OS} TARGETARCH=${params.ARCH}"
				    }
			    }
            }
        }
    }
}