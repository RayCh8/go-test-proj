#!/bin/bash

echo "pre-commit is required, see https://github.com/AmazingTalker/go-amazing#setup-pre-commit-git-hooks for mor information"

pre-commit install -f
pre-commit autoupdate

echo "WELCOME to go-amazing one shot initializer.
Replace rpc name, service name and service name.
Assume your service called \"my-name\", never use words like \"server\", \"service\" or \"rpc\" as a repository name, it will make things confused.
Warning! It's case-sensitive."

read  -p " Please enter a new name, goamazing -> (amazingmyname):" newname
read  -p " Please enter a new name, go-amazing -> (amazing-my-name):" new_name
read  -p " Please enter a new name, GoAmazing -> (AmazingMyName):" NewName

grep --exclude-dir=third_party --exclude-dir=.dockerbuild --exclude-dir=.git -I -rl '.' . | xargs sed -i '' "s/goamazing/${newname}/g"
grep --exclude-dir=third_party --exclude-dir=.dockerbuild --exclude-dir=.git -I -rl '.' . | xargs sed -i '' "s/go-amazing/${new_name}/g"
grep --exclude-dir=third_party --exclude-dir=.dockerbuild --exclude-dir=.git -I -rl '.' . | xargs sed -i '' "s/GoAmazing/${NewName}/g"

read  -p "Do you want to remove unusing files?(y/n)':" yn

if [ $yn = "y" ]
then
    rm -rf doc
    rm ./one_shot_init.sh
    echo "one_shot_init.sh said: bye."
fi

echo "done"
