*** Settings ***
Library     Process
Library     String
Library     OperatingSystem


*** Variables ***
${HONEYPOT} =       localhost:22
${SSH_SERVER} =     localhost:2001


*** Test Cases ***
Basic tests
    Verify same SSH output    ${HONEYPOT}    ${SSH_SERVER}    pwd
    Verify same SSH output    ${HONEYPOT}    ${SSH_SERVER}    whoami
    Verify same SSH output    ${HONEYPOT}    ${SSH_SERVER}    echo "$((1+1))"


*** Keywords ***
Sh
    [Documentation]    Runs a shell command
    [Arguments]    ${cmd}
    ${res}=    Run Process    bash    -c    set -euo pipefail; ${cmd}
    IF    $res.stderr    Log    ${res.stderr}    level=ERROR
    Log Many    stdout=${res.stdout}    stdout=${res.stderr}    rc=${res.rc}
    Should Be Equal As Integers    0    ${res.rc}
    RETURN    ${res}

Verify same SSH output
    [Arguments]    ${honeypot}    ${ssh_server}    ${cmd}    ${user}=root
    ${honeypot}=    Split String    ${honeypot}    :
    ${ssh_server}=    Split String    ${ssh_server}    :

    ${res1}=    Sh
    ...    ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -o "LogLevel=ERROR" ${user}@${honeypot}[0] -p ${honeypot}[1] ${cmd}
    ${res2}=    Sh
    ...    ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -o "LogLevel=ERROR" ${user}@${ssh_server}[0] -p ${ssh_server}[1] ${cmd}

    Should Be Equal As Strings    ${res1.stdout}    ${res2.stdout}
