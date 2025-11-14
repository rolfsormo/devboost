# Core logging functions

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

db_log_info() {
    echo -e "${BLUE}ℹ${NC} $*"
}

db_log_success() {
    echo -e "${GREEN}✓${NC} $*"
}

db_log_warn() {
    echo -e "${YELLOW}⚠${NC} $*"
}

db_log_error() {
    echo -e "${RED}✗${NC} $*" >&2
}

db_log_verbose() {
    if [[ "${DB_VERBOSE:-false}" == "true" ]]; then
        echo -e "${BLUE}[verbose]${NC} $*"
    fi
}

db_die() {
    db_log_error "$@"
    exit 1
}

