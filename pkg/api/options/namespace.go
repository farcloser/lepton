/*
   Copyright Farcloser.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package options

// ListOptions specifies options for `apparmor ls`.
type NamespaceList struct {
	Quiet  bool
	Format string
}

type NamespaceCreate struct {
	Name string
	// Labels are the namespace labels
	Labels map[string]string
}

type NamespaceInspect struct {
	NamesList []string
	// Format the output using the given Go template, e.g, '{{json .}}'
	Format string
}

type NamespaceRemove struct {
	NamesList []string
	// CGroup delete the namespace's cgroup
	CGroup bool
}

type NamespaceUpdate NamespaceCreate
