# YAML config handling via yq

DB_CONFIG_PATH="${DB_CONFIG_PATH:-$HOME/.devboost.yaml}"

db_yaml_get() {
    local path="$1" default="${2-}"
    if [[ -f "$DB_CONFIG_PATH" ]]; then
        if db_command_exists yq; then
            yq -e "$path" "$DB_CONFIG_PATH" 2>/dev/null || echo "$default"
        elif db_command_exists python3 && python3 -c "import yaml" 2>/dev/null; then
            python3 -c "
import yaml
import sys
try:
    with open('$DB_CONFIG_PATH', 'r') as f:
        data = yaml.safe_load(f) or {}
    def get_nested(d, keys):
        for k in keys.split('.'):
            if isinstance(d, dict) and k in d:
                d = d[k]
            else:
                return None
        return d
    result = get_nested(data, '$path')
    print(result if result is not None else '$default')
except:
    print('$default')
"
        else
            echo "$default"
        fi
    else
        echo "$default"
    fi
}

db_yaml_get_list() {
    local path="$1"
    if [[ -f "$DB_CONFIG_PATH" ]]; then
        if db_command_exists yq; then
            yq -e "$path[]" "$DB_CONFIG_PATH" 2>/dev/null | tr '\n' ' ' || echo ""
        else
            echo ""
        fi
    else
        echo ""
    fi
}

