*** Settings ***
Library     Process
Library     String
Library     OperatingSystem


*** Keywords ***
Sh
    [Documentation]    Runs a shell command
    [Arguments]    ${cmd}
    ${res}=    Run Process    bash    -c    set -euo pipefail; ${cmd}
    IF    $res.stderr    Log    ${res.stderr}    level=ERROR
    Log Many    stdout=${res.stdout}    stderr=${res.stderr}    rc=${res.rc}
    Should Be Equal As Integers    0    ${res.rc}
    RETURN    ${res}

Psql
    [Documentation]    Runs a SQL query against the databse
    [Arguments]    ${query}
    ${res}=    Sh    PGPASSWORD=example psql -A -t -h localhost -U postgres -c '${query}'
    RETURN    ${res}

SSH
    [Arguments]    ${server}    ${cmd}    ${user}=root
    ${server}=    Split String    ${server}    :
    ${res}=    Sh
    ...    ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -o "LogLevel=ERROR" ${user}@${server}[0] -p ${server}[1] ${cmd}
    RETURN    ${res}

Verify same SSH output
    [Arguments]    ${server_1}    ${server_2}    ${cmd}    ${user}=root

    ${res1}=    SSH    ${server_1}    ${cmd}    user=${user}
    ${res2}=    SSH    ${server_2}    ${cmd}    user=${user}

    Should Be Equal As Strings    ${res1.stdout}    ${res2.stdout}
