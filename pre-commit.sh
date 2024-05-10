#!/bin/bash

# Функція для встановлення gitleaks на Linux
install_gitleaks_linux() {
    echo "Installing gitleaks on Linux..."
    #curl -sSfL https://raw.githubusercontent.com/zricethezav/gitleaks/master/install.sh | sudo bash
    export GITLEAKS_VERSION=$(wget -qO - "https://api.github.com/repos/gitleaks/gitleaks/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')
    echo $GITLEAKS_VERSION
    wget --no-verbose https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz -O - | tar -zxvf -
#gitleaks_8.18.2_linux_x64.tar.gz

}

# Функція для встановлення gitleaks на macOS
install_gitleaks_macos() {
    echo "Installing gitleaks..."
    brew install gitleaks
}

# Функція для встановлення gitleaks на Windows
install_gitleaks_windows() {
    echo "Gitleaks installation on Windows is not supported by this script."
    echo "Please refer to the gitleaks documentation for installation instructions."
    exit 1
}

# Функція для перевірки, чи gitleaks встановлено
check_gitleaks_installed() {
    if ! command -v gitleaks &> /dev/null; then
        echo "Gitleaks is not installed."
        return 1
    else
        echo "Gitleaks is installed."
        return 0
    fi
}

# Функція для встановлення gitleaks за допомогою git config
enable_gitleaks() {
    git config --local --bool hooks.gitleaks.enabled true
    echo "Gitleaks enabled via git config."
}

# Функція для відключення gitleaks за допомогою git config
disable_gitleaks() {
    git config --local --bool hooks.gitleaks.enabled false
    echo "Gitleaks disabled via git config."
}

# Перевірка операційної системи і виклик відповідної функції для встановлення gitleaks
case "$(uname -s)" in
    Linux*)     install_gitleaks_linux ;;
    Darwin*)    install_gitleaks_macos ;;
    CYGWIN*)    install_gitleaks_windows ;;
    MINGW*)     install_gitleaks_windows ;;
    *)          echo "Unsupported operating system." ;;
esac

# Перевірка, чи потрібно ввімкнути gitleaks через git config
if git config --get hooks.gitleaks.enabled >/dev/null 2>&1; then
    ENABLED=$(git config --get hooks.gitleaks.enabled)
    if [[ "$ENABLED" == "true" ]]; then
        check_gitleaks_installed || exit 1
    else
        echo "Gitleaks is disabled via git config."
        exit 0
    fi
else
    echo "Gitleaks is enabled by default."
    check_gitleaks_installed || exit 1
fi

# Запуск gitleaks для перевірки комміту
gitleaks --verbose

# Перевірка статусу виходу gitleaks і виведення відповідного повідомлення
if [ $? -eq 0 ]; then
    echo "No secrets detected. Commit allowed."
    exit 0
else
    echo "Secrets detected. Commit aborted."
    exit 1
fi
