#!/usr/bin/env bash
# Build script - concatenates all modules into single devboost.sh

set -euo pipefail

OUT="devboost.sh"

{
    # Entry point
    cat devboost.sh.in
    
    # Core framework (order matters)
    echo ""
    echo "# === Core Framework ==="
    cat core/core_log.sh
    echo ""
    cat core/core_os.sh
    echo ""
    cat core/core_yaml.sh
    echo ""
    cat core/core_files.sh
    echo ""
    cat core/core_modules.sh
    echo ""
    cat core/core_main.sh
    
    # Modules (order matters - dependencies first)
    echo ""
    echo "# === Modules ==="
    cat modules/module_pkg.sh
    echo ""
    cat modules/module_znap.sh
    echo ""
    cat modules/module_zsh.sh
    echo ""
    cat modules/module_starship.sh
    echo ""
    cat modules/module_tmux.sh
    echo ""
    cat modules/module_mise.sh
    echo ""
    cat modules/module_direnv.sh
    echo ""
    cat modules/module_git.sh
    echo ""
    cat modules/module_services.sh
    
    # Module registration (must be after all modules are defined)
    echo ""
    echo "# === Module Registration ==="
    cat << 'REGEOF'
db_load_modules() {
    # Register all modules
    db_module_pkg_register
    db_module_znap_register
    db_module_zsh_register
    db_module_starship_register
    db_module_tmux_register
    db_module_mise_register
    db_module_direnv_register
    db_module_git_register
    db_module_services_register
    
    db_log_verbose "Loaded ${#DB_MODULE_NAMES[@]} modules"
}
REGEOF
    
    # Main execution
    echo ""
    echo "# === Main Execution ==="
    cat << 'MAINEOF'
# Run main if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    coreMain "$@"
fi
MAINEOF

} > "$OUT"

chmod +x "$OUT"

echo "Built: $OUT"
echo "Size: $(wc -l < "$OUT") lines"

